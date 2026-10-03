package http

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

	"github.com/go-chi/chi/v5"

	"github.com/ferjmc/god-edu/api/db"
	"github.com/ferjmc/god-edu/api/httpkit"
	courseapp "github.com/ferjmc/god-edu/api/internal/course/application"
	"github.com/ferjmc/god-edu/api/internal/lesson/application"
	"github.com/ferjmc/god-edu/api/internal/lesson/domain"
	"github.com/ferjmc/god-edu/api/storage"
)

// AdminHandler agrupa los endpoints de administración de lecciones y su
// contenido (video, markdown, PDF). Todos están detrás de
// auth.RequireAuth + RequireAdmin (ver newRouter en main.go) — no hay
// bloqueo adicional acá adentro, el middleware ya filtró.
//
// R2 es nil si el server arrancó sin credenciales de R2 configuradas (ver
// config.go): el resto de la API sigue funcionando, pero subir un PDF
// devuelve 503 en vez de un panic por nil pointer.
type AdminHandler struct {
	Courses *courseapp.Service
	Lessons *application.Service
	R2      *storage.R2
}

// maxPDFUploadSize acota cuánto puede pesar un PDF de lección. 25MB es
// generoso para una guía o un apunte de catequesis y chico comparado con
// los 10GB gratis de R2 — evita que una subida (a propósito o por error)
// se coma la cuota rápido.
const maxPDFUploadSize = 25 << 20

// ToContentResponse convierte contenido de dominio a su forma pública —
// la usan tanto el detalle público de lección (lessons.go) como estos
// endpoints de admin.
func ToContentResponse(c domain.LessonContent) ContentResponse {
	return ContentResponse{
		Order:      c.OrderIndex,
		Title:      c.Title,
		Type:       string(c.Type),
		YoutubeURL: c.YoutubeURL,
		PDFURL:     c.PDFURL,
		Body:       c.Body,
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
		if httpkit.HandleNotFound(w, err, "curso no encontrado") {
			return
		}
		log.Printf("admin: obteniendo curso %q: %v", slug, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo obtener el curso")
		return
	}

	var req createLessonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpkit.WriteError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	title, ok := httpkit.RequireNonEmpty(w, req.Title, "título")
	if !ok {
		return
	}
	req.Title = title
	if req.Order < 1 {
		httpkit.WriteError(w, http.StatusBadRequest, "order debe ser 1 o mayor")
		return
	}

	lesson, err := h.Lessons.CreateLesson(r.Context(), domain.Lesson{
		CourseID:   course.ID,
		Title:      req.Title,
		OrderIndex: req.Order,
	})
	if err != nil {
		if errors.Is(err, db.ErrConflict) {
			httpkit.WriteError(w, http.StatusConflict, "ya existe una lección con ese order en este curso")
			return
		}
		log.Printf("admin: creando lección en curso %d: %v", course.ID, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo crear la lección")
		return
	}

	httpkit.WriteJSON(w, http.StatusCreated, summaryResponse{Order: lesson.OrderIndex, Title: lesson.Title})
}

// --- Editar / borrar lección ---

type updateLessonRequest struct {
	Title string `json:"title"`
}

// UpdateLesson renombra una lección. El orden no se puede tocar por acá —
// ver lessonpg.Repository.UpdateTitle.
func (h *AdminHandler) UpdateLesson(w http.ResponseWriter, r *http.Request) {
	lesson, ok := h.resolveLesson(w, r)
	if !ok {
		return
	}

	var req updateLessonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpkit.WriteError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	title, ok := httpkit.RequireNonEmpty(w, req.Title, "título")
	if !ok {
		return
	}
	req.Title = title

	if err := h.Lessons.UpdateTitle(r.Context(), lesson.ID, req.Title); err != nil {
		if httpkit.HandleNotFound(w, err, "lección no encontrada") {
			return
		}
		log.Printf("admin: renombrando lección %d: %v", lesson.ID, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo actualizar la lección")
		return
	}

	httpkit.WriteJSON(w, http.StatusOK, summaryResponse{Order: lesson.OrderIndex, Title: req.Title})
}

// DeleteLesson borra una lección y, en cascada, su contenido y el progreso
// de los usuarios sobre ella. Igual que course/http.AdminHandler.DeleteCourse,
// intenta limpiar sus PDFs de R2 primero, best-effort.
func (h *AdminHandler) DeleteLesson(w http.ResponseWriter, r *http.Request) {
	lesson, ok := h.resolveLesson(w, r)
	if !ok {
		return
	}

	h.cleanupLessonPDFs(r.Context(), lesson.ID)

	if err := h.Lessons.DeleteLesson(r.Context(), lesson.ID); err != nil {
		if httpkit.HandleNotFound(w, err, "lección no encontrada") {
			return
		}
		log.Printf("admin: borrando lección %d: %v", lesson.ID, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo borrar la lección")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// cleanupLessonPDFs borra de R2 los PDFs de una lección, best-effort: si R2
// no está configurado o un borrado puntual falla, lo loguea y sigue — nunca
// bloquea un borrado en la base por un problema de storage. No exportada:
// course/http.AdminHandler.DeleteCourse necesita la misma lógica pero no
// puede importar este paquete (crearía un ciclo con access.go, que este
// paquete sí importa) — tiene su propia copia, ver el comentario ahí.
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
		if c.Type != domain.ContentTypePDF || c.PDFURL == nil {
			continue
		}
		if err := h.R2.DeleteByURL(ctx, *c.PDFURL); err != nil {
			log.Printf("admin: borrando PDF de R2 (%s): %v", *c.PDFURL, err)
		}
	}
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
		httpkit.WriteError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	title, ok := httpkit.RequireNonEmpty(w, req.Title, "título")
	if !ok {
		return
	}
	req.Title = title

	content := domain.LessonContent{LessonID: lesson.ID, Title: req.Title}
	switch req.Type {
	case string(domain.ContentTypeVideo):
		if req.YoutubeURL == nil || strings.TrimSpace(*req.YoutubeURL) == "" {
			httpkit.WriteError(w, http.StatusBadRequest, "youtubeUrl es obligatorio para type=video")
			return
		}
		content.Type = domain.ContentTypeVideo
		content.YoutubeURL = req.YoutubeURL
	case string(domain.ContentTypeMarkdown):
		if req.Body == nil || strings.TrimSpace(*req.Body) == "" {
			httpkit.WriteError(w, http.StatusBadRequest, "body es obligatorio para type=markdown")
			return
		}
		content.Type = domain.ContentTypeMarkdown
		content.Body = req.Body
	case string(domain.ContentTypePDF):
		httpkit.WriteError(w, http.StatusBadRequest, "para PDF usá POST .../content/pdf con el archivo, no este endpoint")
		return
	default:
		httpkit.WriteError(w, http.StatusBadRequest, "type debe ser 'video' o 'markdown'")
		return
	}

	order, err := h.Lessons.NextContentOrder(r.Context(), lesson.ID)
	if err != nil {
		log.Printf("admin: calculando orden de contenido para lección %d: %v", lesson.ID, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo crear el contenido")
		return
	}
	content.OrderIndex = order

	created, err := h.Lessons.CreateContent(r.Context(), content)
	if err != nil {
		log.Printf("admin: creando contenido para lección %d: %v", lesson.ID, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo crear el contenido")
		return
	}

	httpkit.WriteJSON(w, http.StatusCreated, ToContentResponse(created))
}

// filenamePattern reemplaza todo lo que no sea alfanumérico, punto o
// guion por un guion — el nombre original del archivo puede traer
// espacios, acentos o símbolos que no queremos como parte de una key de R2.
var filenamePattern = regexp.MustCompile(`[^a-zA-Z0-9.\-]+`)

// UploadPDFContent sube un PDF a R2 y crea la fila de contenido con la URL
// resultante. multipart/form-data con dos campos: "title" y "file".
func (h *AdminHandler) UploadPDFContent(w http.ResponseWriter, r *http.Request) {
	if h.R2 == nil {
		httpkit.WriteError(w, http.StatusServiceUnavailable, "la subida de PDFs no está configurada (faltan credenciales de R2)")
		return
	}

	lesson, ok := h.resolveLesson(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPDFUploadSize)
	if err := r.ParseMultipartForm(maxPDFUploadSize); err != nil {
		httpkit.WriteError(w, http.StatusBadRequest, "el archivo supera el máximo permitido (25MB) o el form es inválido")
		return
	}

	title, ok := httpkit.RequireNonEmpty(w, r.FormValue("title"), "título")
	if !ok {
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpkit.WriteError(w, http.StatusBadRequest, "falta el archivo (campo 'file')")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType != "application/pdf" {
		httpkit.WriteError(w, http.StatusBadRequest, "el archivo debe ser un PDF")
		return
	}

	order, err := h.Lessons.NextContentOrder(r.Context(), lesson.ID)
	if err != nil {
		log.Printf("admin: calculando orden de contenido para lección %d: %v", lesson.ID, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo subir el archivo")
		return
	}

	safeName := filenamePattern.ReplaceAllString(header.Filename, "-")
	key := fmt.Sprintf("lessons/%d/%d-%s", lesson.ID, order, safeName)

	pdfURL, err := h.R2.Upload(r.Context(), key, file, contentType)
	if err != nil {
		log.Printf("admin: subiendo PDF a R2 (lección %d): %v", lesson.ID, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo subir el archivo")
		return
	}

	created, err := h.Lessons.CreateContent(r.Context(), domain.LessonContent{
		LessonID:   lesson.ID,
		Title:      title,
		OrderIndex: order,
		Type:       domain.ContentTypePDF,
		PDFURL:     &pdfURL,
	})
	if err != nil {
		log.Printf("admin: guardando contenido PDF para lección %d: %v", lesson.ID, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "el archivo se subió pero no se pudo guardar el registro")
		return
	}

	httpkit.WriteJSON(w, http.StatusCreated, ToContentResponse(created))
}

// resolveLesson lee slug + order de la URL (.../courses/{slug}/lessons/{order}/...)
// y devuelve la lección correspondiente, o escribe la respuesta de error y
// devuelve ok=false. Común a CreateContent y UploadPDFContent.
func (h *AdminHandler) resolveLesson(w http.ResponseWriter, r *http.Request) (domain.Lesson, bool) {
	slug := chi.URLParam(r, "slug")
	order, err := strconv.Atoi(chi.URLParam(r, "order"))
	if err != nil || order < 1 {
		httpkit.WriteError(w, http.StatusBadRequest, "número de lección inválido")
		return domain.Lesson{}, false
	}

	course, err := h.Courses.GetBySlugAny(r.Context(), slug)
	if err != nil {
		if httpkit.HandleNotFound(w, err, "curso no encontrado") {
			return domain.Lesson{}, false
		}
		log.Printf("admin: obteniendo curso %q: %v", slug, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo obtener el curso")
		return domain.Lesson{}, false
	}

	lesson, err := h.Lessons.GetByCourseAndOrder(r.Context(), course.ID, order)
	if err != nil {
		if httpkit.HandleNotFound(w, err, "lección no encontrada") {
			return domain.Lesson{}, false
		}
		log.Printf("admin: obteniendo lección %d/%d: %v", course.ID, order, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo obtener la lección")
		return domain.Lesson{}, false
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
		httpkit.WriteError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	title, ok := httpkit.RequireNonEmpty(w, req.Title, "título")
	if !ok {
		return
	}
	req.Title = title

	switch content.Type {
	case domain.ContentTypeVideo:
		if req.YoutubeURL == nil || strings.TrimSpace(*req.YoutubeURL) == "" {
			httpkit.WriteError(w, http.StatusBadRequest, "youtubeUrl es obligatorio para este contenido")
			return
		}
	case domain.ContentTypeMarkdown:
		if req.Body == nil || strings.TrimSpace(*req.Body) == "" {
			httpkit.WriteError(w, http.StatusBadRequest, "body es obligatorio para este contenido")
			return
		}
	case domain.ContentTypePDF:
		// Solo el título es editable acá; youtube_url/body quedan en null
		// igual que hoy, no hace falta validar nada más.
		req.YoutubeURL = nil
		req.Body = nil
	}

	if err := h.Lessons.UpdateContent(r.Context(), content.ID, req.Title, req.YoutubeURL, req.Body); err != nil {
		if httpkit.HandleNotFound(w, err, "contenido no encontrado") {
			return
		}
		log.Printf("admin: actualizando contenido %d: %v", content.ID, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo actualizar el contenido")
		return
	}

	httpkit.WriteJSON(w, http.StatusOK, ContentResponse{
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

	if h.R2 != nil && content.Type == domain.ContentTypePDF && content.PDFURL != nil {
		if err := h.R2.DeleteByURL(r.Context(), *content.PDFURL); err != nil {
			log.Printf("admin: borrando PDF de R2 (%s): %v", *content.PDFURL, err)
		}
	}

	if err := h.Lessons.DeleteContent(r.Context(), content.ID); err != nil {
		if httpkit.HandleNotFound(w, err, "contenido no encontrado") {
			return
		}
		log.Printf("admin: borrando contenido %d: %v", content.ID, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo borrar el contenido")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// resolveContent lee slug + order de lección + order de contenido de la URL
// (.../courses/{slug}/lessons/{order}/content/{contentOrder}) y devuelve la
// lección y la pieza de contenido correspondientes, o escribe la respuesta
// de error y devuelve ok=false. Común a UpdateContent y DeleteContent.
func (h *AdminHandler) resolveContent(w http.ResponseWriter, r *http.Request) (domain.Lesson, domain.LessonContent, bool) {
	lesson, ok := h.resolveLesson(w, r)
	if !ok {
		return domain.Lesson{}, domain.LessonContent{}, false
	}

	contentOrder, err := strconv.Atoi(chi.URLParam(r, "contentOrder"))
	if err != nil || contentOrder < 1 {
		httpkit.WriteError(w, http.StatusBadRequest, "número de contenido inválido")
		return domain.Lesson{}, domain.LessonContent{}, false
	}

	content, err := h.Lessons.GetContentByLessonAndOrder(r.Context(), lesson.ID, contentOrder)
	if err != nil {
		if httpkit.HandleNotFound(w, err, "contenido no encontrado") {
			return domain.Lesson{}, domain.LessonContent{}, false
		}
		log.Printf("admin: obteniendo contenido %d/%d: %v", lesson.ID, contentOrder, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo obtener el contenido")
		return domain.Lesson{}, domain.LessonContent{}, false
	}

	return lesson, content, true
}
