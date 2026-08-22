package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	Users   *db.UserRepo
	Courses *db.CourseRepo
	Lessons *db.LessonRepo
	R2      *storage.R2
}

// slugPattern es deliberadamente estricto: minúsculas, números y guiones.
// Un slug raro (espacios, mayúsculas, unicode) después complica armar URLs
// y comparar rutas — mejor rechazarlo acá que arrastrarlo por todo el sitio.
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// --- Listar / ver curso ---

// adminCourseResponse es la representación de un curso para el panel
// admin: a diferencia de courseResponse (la pública, en courses.go), lleva
// todo lo que un admin necesita para decidir qué tocar — estado de
// publicación, roles con acceso restringido y fecha de alta.
type adminCourseResponse struct {
	ID           int64             `json:"id"`
	Title        string            `json:"title"`
	Slug         string            `json:"slug"`
	Description  *string           `json:"description"`
	Published    bool              `json:"published"`
	VisibleRoles []models.UserRole `json:"visibleRoles"`
	CreatedAt    time.Time         `json:"createdAt"`
}

func toAdminCourseResponse(c models.Course) adminCourseResponse {
	return adminCourseResponse{
		ID:           c.ID,
		Title:        c.Title,
		Slug:         c.Slug,
		Description:  c.Description,
		Published:    c.Published,
		VisibleRoles: c.VisibleRoles,
		CreatedAt:    c.CreatedAt,
	}
}

// ListCourses devuelve todos los cursos (publicados y en borrador) para el
// panel admin. A diferencia de CourseHandler.List (la vidriera pública),
// acá sí se ven los borradores — es la única forma de retomar un curso a
// medio cargar.
func (h *AdminHandler) ListCourses(w http.ResponseWriter, r *http.Request) {
	courses, err := h.Courses.ListAll(r.Context())
	if err != nil {
		log.Printf("admin: listando cursos: %v", err)
		writeError(w, http.StatusInternalServerError, "no se pudieron obtener los cursos")
		return
	}

	response := make([]adminCourseResponse, len(courses))
	for i, c := range courses {
		response[i] = toAdminCourseResponse(c)
	}

	writeJSON(w, http.StatusOK, response)
}

// adminLessonResponse es una lección con su contenido completo, tal como
// la necesita la pantalla de edición del panel admin.
type adminLessonResponse struct {
	Order   int                     `json:"order"`
	Title   string                  `json:"title"`
	Content []lessonContentResponse `json:"content"`
}

type adminCourseDetailResponse struct {
	adminCourseResponse
	Lessons []adminLessonResponse `json:"lessons"`
}

// CourseDetail devuelve un curso (publicado o en borrador) con toda su
// currícula: cada lección y su contenido (video, PDFs, markdown), en el
// orden en que se cargaron. Es el endpoint que arma la pantalla de edición
// completa de un curso en el panel admin de una sola pasada.
func (h *AdminHandler) CourseDetail(w http.ResponseWriter, r *http.Request) {
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

	lessons, err := h.Lessons.ListAllByCourse(r.Context(), course.ID)
	if err != nil {
		log.Printf("admin: listando lecciones del curso %d: %v", course.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudieron obtener las lecciones")
		return
	}

	lessonsResponse := make([]adminLessonResponse, len(lessons))
	for i, l := range lessons {
		content, err := h.Lessons.ListContent(r.Context(), l.ID)
		if err != nil {
			log.Printf("admin: listando contenido de lección %d: %v", l.ID, err)
			writeError(w, http.StatusInternalServerError, "no se pudo obtener el contenido de una lección")
			return
		}

		contentResponse := make([]lessonContentResponse, len(content))
		for j, c := range content {
			contentResponse[j] = lessonContentResponse{
				Order:      c.OrderIndex,
				Title:      c.Title,
				Type:       string(c.Type),
				YoutubeURL: c.YoutubeURL,
				PDFURL:     c.PDFURL,
				Body:       c.Body,
			}
		}

		lessonsResponse[i] = adminLessonResponse{
			Order:   l.OrderIndex,
			Title:   l.Title,
			Content: contentResponse,
		}
	}

	writeJSON(w, http.StatusOK, adminCourseDetailResponse{
		adminCourseResponse: toAdminCourseResponse(course),
		Lessons:             lessonsResponse,
	})
}

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

// --- Editar / borrar curso ---

type updateCourseDetailsRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
}

// UpdateCourseDetails edita título y descripción de un curso. No toca el
// slug (ver CourseRepo.UpdateDetails) ni el estado de publicación — eso
// sigue siendo SetPublished, a propósito, para no mezclar "editar contenido"
// con "hacerlo visible".
func (h *AdminHandler) UpdateCourseDetails(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var req updateCourseDetailsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "el título es obligatorio")
		return
	}

	if err := h.Courses.UpdateDetails(r.Context(), slug, req.Title, req.Description); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "curso no encontrado")
			return
		}
		log.Printf("admin: actualizando curso %q: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "no se pudo actualizar el curso")
		return
	}

	course, err := h.Courses.GetBySlugAny(r.Context(), slug)
	if err != nil {
		log.Printf("admin: releyendo curso %q tras actualizar: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "el curso se actualizó pero no se pudo confirmar")
		return
	}

	writeJSON(w, http.StatusOK, toAdminCourseResponse(course))
}

// DeleteCourse borra un curso completo: en cascada (ver migrations) se
// llevan puesto sus lecciones, contenido, inscripciones y progreso. Antes
// de borrar, intenta limpiar de R2 los PDFs de todas sus lecciones — si
// alguno falla, lo loguea pero no aborta el borrado: un PDF huérfano en R2
// (dentro de la capa gratis de 10GB) es preferible a un curso que no se
// puede borrar por un problema de red pasajero.
func (h *AdminHandler) DeleteCourse(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	course, err := h.Courses.GetBySlugAny(r.Context(), slug)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "curso no encontrado")
			return
		}
		log.Printf("admin: obteniendo curso %q para borrar: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "no se pudo borrar el curso")
		return
	}

	lessons, err := h.Lessons.ListAllByCourse(r.Context(), course.ID)
	if err != nil {
		log.Printf("admin: listando lecciones del curso %d antes de borrar: %v", course.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudo borrar el curso")
		return
	}
	for _, lesson := range lessons {
		h.cleanupLessonPDFs(r.Context(), lesson.ID)
	}

	if err := h.Courses.Delete(r.Context(), slug); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "curso no encontrado")
			return
		}
		log.Printf("admin: borrando curso %q: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "no se pudo borrar el curso")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// cleanupLessonPDFs borra de R2 los PDFs de una lección, best-effort: si R2
// no está configurado o un borrado puntual falla, lo loguea y sigue — nunca
// bloquea un borrado en la base por un problema de storage.
func (h *AdminHandler) cleanupLessonPDFs(ctx context.Context, lessonID int64) {
	if h.R2 == nil {
		return
	}

	content, err := h.Lessons.ListContent(ctx, lessonID)
	if err != nil {
		log.Printf("admin: listando contenido de lección %d para limpiar R2: %v", lessonID, err)
		return
	}

	for _, c := range content {
		if c.Type != models.ContentTypePDF || c.PDFURL == nil {
			continue
		}
		if err := h.R2.DeleteByURL(ctx, *c.PDFURL); err != nil {
			log.Printf("admin: borrando PDF de R2 (%s): %v", *c.PDFURL, err)
		}
	}
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

// --- Editar / borrar lección ---

type updateLessonRequest struct {
	Title string `json:"title"`
}

// UpdateLesson renombra una lección. El orden no se puede tocar por acá —
// ver LessonRepo.UpdateTitle.
func (h *AdminHandler) UpdateLesson(w http.ResponseWriter, r *http.Request) {
	lesson, ok := h.resolveLesson(w, r)
	if !ok {
		return
	}

	var req updateLessonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "el título es obligatorio")
		return
	}

	if err := h.Lessons.UpdateTitle(r.Context(), lesson.ID, req.Title); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "lección no encontrada")
			return
		}
		log.Printf("admin: renombrando lección %d: %v", lesson.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudo actualizar la lección")
		return
	}

	writeJSON(w, http.StatusOK, lessonSummaryResponse{Order: lesson.OrderIndex, Title: req.Title})
}

// DeleteLesson borra una lección y, en cascada, su contenido y el progreso
// de los usuarios sobre ella. Igual que DeleteCourse, intenta limpiar sus
// PDFs de R2 primero, best-effort.
func (h *AdminHandler) DeleteLesson(w http.ResponseWriter, r *http.Request) {
	lesson, ok := h.resolveLesson(w, r)
	if !ok {
		return
	}

	h.cleanupLessonPDFs(r.Context(), lesson.ID)

	if err := h.Lessons.DeleteLesson(r.Context(), lesson.ID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "lección no encontrada")
			return
		}
		log.Printf("admin: borrando lección %d: %v", lesson.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudo borrar la lección")
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
		Order:      created.OrderIndex,
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
		Order:  created.OrderIndex,
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

// --- Editar / borrar contenido ---

type updateContentRequest struct {
	Title      string  `json:"title"`
	YoutubeURL *string `json:"youtubeUrl"`
	Body       *string `json:"body"`
}

// UpdateContent edita el título y el campo específico de tipo de una pieza
// de contenido ya cargada. El tipo no se puede cambiar (un video no se
// convierte en markdown) ni, para PDF, el archivo — para reemplazar un PDF
// hay que borrar esta pieza y subir una nueva (ver UploadPDFContent).
func (h *AdminHandler) UpdateContent(w http.ResponseWriter, r *http.Request) {
	_, content, ok := h.resolveContent(w, r)
	if !ok {
		return
	}

	var req updateContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "el título es obligatorio")
		return
	}

	switch content.Type {
	case models.ContentTypeVideo:
		if req.YoutubeURL == nil || strings.TrimSpace(*req.YoutubeURL) == "" {
			writeError(w, http.StatusBadRequest, "youtubeUrl es obligatorio para este contenido")
			return
		}
	case models.ContentTypeMarkdown:
		if req.Body == nil || strings.TrimSpace(*req.Body) == "" {
			writeError(w, http.StatusBadRequest, "body es obligatorio para este contenido")
			return
		}
	case models.ContentTypePDF:
		// Solo el título es editable acá; youtube_url/body quedan en null
		// igual que hoy, no hace falta validar nada más.
		req.YoutubeURL = nil
		req.Body = nil
	}

	if err := h.Lessons.UpdateContent(r.Context(), content.ID, req.Title, req.YoutubeURL, req.Body); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contenido no encontrado")
			return
		}
		log.Printf("admin: actualizando contenido %d: %v", content.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudo actualizar el contenido")
		return
	}

	writeJSON(w, http.StatusOK, lessonContentResponse{
		Order:      content.OrderIndex,
		Title:      req.Title,
		Type:       string(content.Type),
		YoutubeURL: req.YoutubeURL,
		PDFURL:     content.PDFURL,
		Body:       req.Body,
	})
}

// DeleteContent borra una pieza de contenido puntual. Si es un PDF, intenta
// borrarlo de R2 primero, best-effort (ver cleanupLessonPDFs).
func (h *AdminHandler) DeleteContent(w http.ResponseWriter, r *http.Request) {
	_, content, ok := h.resolveContent(w, r)
	if !ok {
		return
	}

	if h.R2 != nil && content.Type == models.ContentTypePDF && content.PDFURL != nil {
		if err := h.R2.DeleteByURL(r.Context(), *content.PDFURL); err != nil {
			log.Printf("admin: borrando PDF de R2 (%s): %v", *content.PDFURL, err)
		}
	}

	if err := h.Lessons.DeleteContent(r.Context(), content.ID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contenido no encontrado")
			return
		}
		log.Printf("admin: borrando contenido %d: %v", content.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudo borrar el contenido")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// resolveContent lee slug + order de lección + order de contenido de la URL
// (.../courses/{slug}/lessons/{order}/content/{contentOrder}) y devuelve la
// lección y la pieza de contenido correspondientes, o escribe la respuesta
// de error y devuelve ok=false. Común a UpdateContent y DeleteContent.
func (h *AdminHandler) resolveContent(w http.ResponseWriter, r *http.Request) (models.Lesson, models.LessonContent, bool) {
	lesson, ok := h.resolveLesson(w, r)
	if !ok {
		return models.Lesson{}, models.LessonContent{}, false
	}

	contentOrder, err := strconv.Atoi(chi.URLParam(r, "contentOrder"))
	if err != nil || contentOrder < 1 {
		writeError(w, http.StatusBadRequest, "número de contenido inválido")
		return models.Lesson{}, models.LessonContent{}, false
	}

	content, err := h.Lessons.GetContentByLessonAndOrder(r.Context(), lesson.ID, contentOrder)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contenido no encontrado")
			return models.Lesson{}, models.LessonContent{}, false
		}
		log.Printf("admin: obteniendo contenido %d/%d: %v", lesson.ID, contentOrder, err)
		writeError(w, http.StatusInternalServerError, "no se pudo obtener el contenido")
		return models.Lesson{}, models.LessonContent{}, false
	}

	return lesson, content, true
}

// --- Listar / ver usuarios ---

// adminUserResponse es la representación de un usuario para el panel
// admin: lleva todo lo que un admin necesita para decidir qué tocar — estado de
// usuario, roles y fecha de alta.
type adminUserResponse struct {
	ID            int64             `json:"id"`
	Email         string            `json:"email"`
	Name          string            `json:"name"`
	AuthProvider  string            `json:"auth_provider"`
	EmailVerified bool              `json:"email_verified"`
	Role          string            `json:"role"`
	CreatedAt     time.Time         `json:"createdAt"`
}

func toAdminUserResponse(u models.User) adminUserResponse {
	return adminUserResponse{
		ID:             u.ID,
		Email: 			u.Email,
		Name:           u.Name,
		AuthProvider:   string(u.AuthProvider),
		EmailVerified:  u.EmailVerified,
		Role: 			string(u.Role),
		CreatedAt:    	u.CreatedAt,
	}
}

// ListUsers devuelve todos los usuarios (activos e inactivos) para el
// panel admin. 
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Users.ListAll(r.Context())
	if err != nil {
		log.Printf("admin: listando usuarios: %v", err)
		writeError(w, http.StatusInternalServerError, "no se pudieron obtener los usuarios")
		return
	}

	response := make([]adminUserResponse, len(users))
	for i, u := range users {
		response[i] = toAdminUserResponse(u)
	}

	writeJSON(w, http.StatusOK, response)
}
