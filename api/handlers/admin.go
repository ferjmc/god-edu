package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/ferjmc/god-edu/api/db"
	"github.com/ferjmc/god-edu/api/models"
	"github.com/ferjmc/god-edu/api/storage"
)

// maxPDFUploadSize acota cuánto puede pesar un PDF de lección. 25MB es
// generoso para una guía o un apunte de catequesis y chico comparado con
// los 10GB gratis de R2 — evita que una subida (a propósito o por error)
// se coma la cuota rápido.
const maxPDFUploadSize = 25 << 20

// AdminHandler agrupa los endpoints de carga de contenido: crear cursos,
// lecciones y su contenido (video, markdown, PDF). Todos están detrás de
// auth.RequireAuth + RequireAdmin (ver newRouter en main.go) — no hay
// bloqueo adicional acá adentro, el middleware ya filtró.
//
// R2 es nil si el server arrancó sin credenciales de R2 configuradas (ver
// config.go): el resto de la API sigue funcionando, pero subir un PDF
// devuelve 503 en vez de un panic por nil pointer.
type AdminHandler struct {
	Courses *db.CourseRepo
	Lessons *db.LessonRepo
	R2      *storage.R2
}

// slugPattern es deliberadamente estricto: minúsculas, números y guiones.
// Un slug raro (espacios, mayúsculas, unicode) después complica armar URLs
// y comparar rutas — mejor rechazarlo acá que arrastrarlo por todo el sitio.
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// --- Crear curso ---

type createCourseRequest struct {
	Title       string  `json:"title"`
	Slug        string  `json:"slug"`
	Description *string `json:"description"`
}

// CreateCourse crea un curso nuevo, siempre en borrador. Publicarlo es un
// paso aparte (ver SetPublished) para poder cargar lecciones y contenido
// con tranquilidad antes de que sea visible.
func (h *AdminHandler) CreateCourse(w http.ResponseWriter, r *http.Request) {
	var req createCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Slug = strings.TrimSpace(req.Slug)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "el título es obligatorio")
		return
	}
	if !slugPattern.MatchString(req.Slug) {
		writeError(w, http.StatusBadRequest, "el slug debe ser minúsculas, números y guiones (ej. mi-curso-nuevo)")
		return
	}

	course, err := h.Courses.Create(r.Context(), models.Course{
		Title:       req.Title,
		Slug:        req.Slug,
		Description: req.Description,
	})
	if err != nil {
		if errors.Is(err, db.ErrConflict) {
			writeError(w, http.StatusConflict, "ya existe un curso con ese slug")
			return
		}
		log.Printf("admin: creando curso: %v", err)
		writeError(w, http.StatusInternalServerError, "no se pudo crear el curso")
		return
	}

	writeJSON(w, http.StatusCreated, toCourseResponse(course))
}

// --- Publicar / despublicar curso ---

type setPublishedRequest struct {
	Published bool `json:"published"`
}

// SetPublished cambia el estado de publicación de un curso por su slug.
func (h *AdminHandler) SetPublished(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var req setPublishedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}

	if err := h.Courses.SetPublished(r.Context(), slug, req.Published); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "curso no encontrado")
			return
		}
		log.Printf("admin: publicando curso %q: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "no se pudo actualizar el curso")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Crear lección ---

type createLessonRequest struct {
	Title string `json:"title"`
	Order int    `json:"order"`
}

// CreateLesson agrega una lección a un curso (existente o en borrador —
// usa GetBySlugAny, no GetBySlug, para no bloquear la carga de contenido
// de un curso que todavía no se publicó).
func (h *AdminHandler) CreateLesson(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	course, err := h.Courses.GetBySlugAny(r.Context(), slug)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "curso no encontrado")
			return
		}
		log.Printf("admin: obteniendo curso %q: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "no se pudo obtener el curso")
		return
	}

	var req createLessonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "el título es obligatorio")
		return
	}
	if req.Order < 1 {
		writeError(w, http.StatusBadRequest, "order debe ser 1 o mayor")
		return
	}

	lesson, err := h.Lessons.CreateLesson(r.Context(), models.Lesson{
		CourseID:   course.ID,
		Title:      req.Title,
		OrderIndex: req.Order,
	})
	if err != nil {
		if errors.Is(err, db.ErrConflict) {
			writeError(w, http.StatusConflict, "ya existe una lección con ese order en este curso")
			return
		}
		log.Printf("admin: creando lección en curso %d: %v", course.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudo crear la lección")
		return
	}

	writeJSON(w, http.StatusCreated, lessonSummaryResponse{Order: lesson.OrderIndex, Title: lesson.Title})
}

// --- Crear contenido: video o markdown (JSON) ---

type createContentRequest struct {
	Title      string  `json:"title"`
	Type       string  `json:"type"`
	YoutubeURL *string `json:"youtubeUrl"`
	Body       *string `json:"body"`
}

// CreateContent agrega una pieza de contenido de tipo video o markdown a
// una lección. Para PDF ver UploadPDFContent — un body JSON no es un buen
// lugar para mandar el archivo, así que es un endpoint aparte.
func (h *AdminHandler) CreateContent(w http.ResponseWriter, r *http.Request) {
	lesson, ok := h.resolveLesson(w, r)
	if !ok {
		return
	}

	var req createContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "el título es obligatorio")
		return
	}

	content := models.LessonContent{LessonID: lesson.ID, Title: req.Title}
	switch req.Type {
	case string(models.ContentTypeVideo):
		if req.YoutubeURL == nil || strings.TrimSpace(*req.YoutubeURL) == "" {
			writeError(w, http.StatusBadRequest, "youtubeUrl es obligatorio para type=video")
			return
		}
		content.Type = models.ContentTypeVideo
		content.YoutubeURL = req.YoutubeURL
	case string(models.ContentTypeMarkdown):
		if req.Body == nil || strings.TrimSpace(*req.Body) == "" {
			writeError(w, http.StatusBadRequest, "body es obligatorio para type=markdown")
			return
		}
		content.Type = models.ContentTypeMarkdown
		content.Body = req.Body
	case string(models.ContentTypePDF):
		writeError(w, http.StatusBadRequest, "para PDF usá POST .../content/pdf con el archivo, no este endpoint")
		return
	default:
		writeError(w, http.StatusBadRequest, "type debe ser 'video' o 'markdown'")
		return
	}

	order, err := h.Lessons.NextContentOrder(r.Context(), lesson.ID)
	if err != nil {
		log.Printf("admin: calculando orden de contenido para lección %d: %v", lesson.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudo crear el contenido")
		return
	}
	content.OrderIndex = order

	created, err := h.Lessons.CreateContent(r.Context(), content)
	if err != nil {
		log.Printf("admin: creando contenido para lección %d: %v", lesson.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudo crear el contenido")
		return
	}

	writeJSON(w, http.StatusCreated, lessonContentResponse{
		Title:      created.Title,
		Type:       string(created.Type),
		YoutubeURL: created.YoutubeURL,
		Body:       created.Body,
	})
}

// filenamePattern reemplaza todo lo que no sea alfanumérico, punto o
// guion por un guion — el nombre original del archivo puede traer
// espacios, acentos o símbolos que no queremos como parte de una key de R2.
var filenamePattern = regexp.MustCompile(`[^a-zA-Z0-9.\-]+`)

// UploadPDFContent sube un PDF a R2 y crea la fila de contenido con la URL
// resultante. multipart/form-data con dos campos: "title" y "file".
func (h *AdminHandler) UploadPDFContent(w http.ResponseWriter, r *http.Request) {
	if h.R2 == nil {
		writeError(w, http.StatusServiceUnavailable, "la subida de PDFs no está configurada (faltan credenciales de R2)")
		return
	}

	lesson, ok := h.resolveLesson(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPDFUploadSize)
	if err := r.ParseMultipartForm(maxPDFUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "el archivo supera el máximo permitido (25MB) o el form es inválido")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		writeError(w, http.StatusBadRequest, "el título es obligatorio")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "falta el archivo (campo 'file')")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType != "application/pdf" {
		writeError(w, http.StatusBadRequest, "el archivo debe ser un PDF")
		return
	}

	order, err := h.Lessons.NextContentOrder(r.Context(), lesson.ID)
	if err != nil {
		log.Printf("admin: calculando orden de contenido para lección %d: %v", lesson.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudo subir el archivo")
		return
	}

	safeName := filenamePattern.ReplaceAllString(header.Filename, "-")
	key := fmt.Sprintf("lessons/%d/%d-%s", lesson.ID, order, safeName)

	pdfURL, err := h.R2.Upload(r.Context(), key, file, contentType)
	if err != nil {
		log.Printf("admin: subiendo PDF a R2 (lección %d): %v", lesson.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudo subir el archivo")
		return
	}

	created, err := h.Lessons.CreateContent(r.Context(), models.LessonContent{
		LessonID:   lesson.ID,
		Title:      title,
		OrderIndex: order,
		Type:       models.ContentTypePDF,
		PDFURL:     &pdfURL,
	})
	if err != nil {
		log.Printf("admin: guardando contenido PDF para lección %d: %v", lesson.ID, err)
		writeError(w, http.StatusInternalServerError, "el archivo se subió pero no se pudo guardar el registro")
		return
	}

	writeJSON(w, http.StatusCreated, lessonContentResponse{
		Title:  created.Title,
		Type:   string(created.Type),
		PDFURL: created.PDFURL,
	})
}

// resolveLesson lee slug + order de la URL (.../courses/{slug}/lessons/{order}/...)
// y devuelve la lección correspondiente, o escribe la respuesta de error y
// devuelve ok=false. Común a CreateContent y UploadPDFContent.
func (h *AdminHandler) resolveLesson(w http.ResponseWriter, r *http.Request) (models.Lesson, bool) {
	slug := chi.URLParam(r, "slug")
	order, err := strconv.Atoi(chi.URLParam(r, "order"))
	if err != nil || order < 1 {
		writeError(w, http.StatusBadRequest, "número de lección inválido")
		return models.Lesson{}, false
	}

	course, err := h.Courses.GetBySlugAny(r.Context(), slug)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "curso no encontrado")
			return models.Lesson{}, false
		}
		log.Printf("admin: obteniendo curso %q: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "no se pudo obtener el curso")
		return models.Lesson{}, false
	}

	lesson, err := h.Lessons.GetByCourseAndOrder(r.Context(), course.ID, order)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "lección no encontrada")
			return models.Lesson{}, false
		}
		log.Printf("admin: obteniendo lección %d/%d: %v", course.ID, order, err)
		writeError(w, http.StatusInternalServerError, "no se pudo obtener la lección")
		return models.Lesson{}, false
	}

	return lesson, true
}
