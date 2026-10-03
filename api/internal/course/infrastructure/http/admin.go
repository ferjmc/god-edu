package http

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ferjmc/god-edu/api/db"
	"github.com/ferjmc/god-edu/api/httpkit"
	"github.com/ferjmc/god-edu/api/internal/course/application"
	"github.com/ferjmc/god-edu/api/internal/course/domain"
	lessonapp "github.com/ferjmc/god-edu/api/internal/lesson/application"
	lessondomain "github.com/ferjmc/god-edu/api/internal/lesson/domain"
	userdomain "github.com/ferjmc/god-edu/api/internal/user/domain"
	"github.com/ferjmc/god-edu/api/storage"
)

// AdminHandler agrupa los endpoints de administración de cursos: crear,
// editar, publicar, borrar, y ver la currícula completa (lecciones +
// contenido) para la pantalla de edición. Todos están detrás de
// auth.RequireAuth + RequireAdmin (ver newRouter en main.go) — no hay
// bloqueo adicional acá adentro, el middleware ya filtró.
//
// R2 es nil si el server arrancó sin credenciales de R2 configuradas (ver
// config.go): el resto de la API sigue funcionando, pero limpiar PDFs al
// borrar un curso simplemente no hace nada (ver cleanupLessonPDFs).
type AdminHandler struct {
	Courses *application.Service
	Lessons *lessonapp.Service
	R2      *storage.R2
}

// slugPattern es deliberadamente estricto: minúsculas, números y guiones.
// Un slug raro (espacios, mayúsculas, unicode) después complica armar URLs
// y comparar rutas — mejor rechazarlo acá que arrastrarlo por todo el sitio.
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// reservedSlugs son slugs que colisionarían con una ruta estática de
// /courses (hoy solo "mine", ver GET /courses/mine en main.go): un curso
// con ese slug quedaría inalcanzable por /courses/{slug} para siempre.
var reservedSlugs = map[string]bool{"mine": true}

// --- Listar / ver curso ---

// adminCourseResponse es la representación de un curso para el panel
// admin: a diferencia de courseResponse (la pública, en courses.go), lleva
// todo lo que un admin necesita para decidir qué tocar — estado de
// publicación, roles con acceso restringido y fecha de alta.
type adminCourseResponse struct {
	ID                 int64                 `json:"id"`
	Title              string                `json:"title"`
	Slug               string                `json:"slug"`
	Description        *string               `json:"description"`
	Published          bool                  `json:"published"`
	CertificateEnabled bool                  `json:"certificateEnabled"`
	VisibleRoles       []userdomain.UserRole `json:"visibleRoles"`
	CreatedAt          time.Time             `json:"createdAt"`
}

func toAdminCourseResponse(c domain.Course) adminCourseResponse {
	return adminCourseResponse{
		ID:                 c.ID,
		Title:              c.Title,
		Slug:               c.Slug,
		Description:        c.Description,
		Published:          c.Published,
		CertificateEnabled: c.CertificateEnabled,
		VisibleRoles:       c.VisibleRoles,
		CreatedAt:          c.CreatedAt,
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
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudieron obtener los cursos")
		return
	}

	response := make([]adminCourseResponse, len(courses))
	for i, c := range courses {
		response[i] = toAdminCourseResponse(c)
	}

	httpkit.WriteJSON(w, http.StatusOK, response)
}

// contentResponse es el contenido de una lección tal como lo necesita la
// pantalla de edición del panel admin, anidado dentro de CourseDetail. Es
// una copia deliberada de la misma forma que expone
// lesson/infrastructure/http (no se importa desde ahí): ese paquete
// depende de este (ver access.go, ResolveVisibleCourse), así que esta
// carpeta NO puede importar lesson/http sin crear un ciclo — duplicar un
// DTO de 6 campos es más simple que introducir un tercer paquete solo para
// compartirlo.
type contentResponse struct {
	Order      int     `json:"order"`
	Title      string  `json:"title"`
	Type       string  `json:"type"`
	YoutubeURL *string `json:"youtubeUrl,omitempty"`
	PDFURL     *string `json:"pdfUrl,omitempty"`
	Body       *string `json:"body,omitempty"`
}

// adminLessonResponse es una lección con su contenido completo, tal como
// la necesita la pantalla de edición del panel admin.
type adminLessonResponse struct {
	Order   int               `json:"order"`
	Title   string            `json:"title"`
	Content []contentResponse `json:"content"`
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
		if httpkit.HandleNotFound(w, err, "curso no encontrado") {
			return
		}
		log.Printf("admin: obteniendo curso %q: %v", slug, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo obtener el curso")
		return
	}

	lessons, err := h.Lessons.ListAllByCourse(r.Context(), course.ID)
	if err != nil {
		log.Printf("admin: listando lecciones del curso %d: %v", course.ID, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudieron obtener las lecciones")
		return
	}

	lessonsResponse := make([]adminLessonResponse, len(lessons))
	for i, l := range lessons {
		content, err := h.Lessons.ListContent(r.Context(), l.ID)
		if err != nil {
			log.Printf("admin: listando contenido de lección %d: %v", l.ID, err)
			httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo obtener el contenido de una lección")
			return
		}

		contentRes := make([]contentResponse, len(content))
		for j, c := range content {
			contentRes[j] = contentResponse{
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
			Content: contentRes,
		}
	}

	httpkit.WriteJSON(w, http.StatusOK, adminCourseDetailResponse{
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
		httpkit.WriteError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}

	title, ok := httpkit.RequireNonEmpty(w, req.Title, "título")
	if !ok {
		return
	}
	req.Title = title
	req.Slug = strings.TrimSpace(req.Slug)
	if !slugPattern.MatchString(req.Slug) {
		httpkit.WriteError(w, http.StatusBadRequest, "el slug debe ser minúsculas, números y guiones (ej. mi-curso-nuevo)")
		return
	}
	if reservedSlugs[req.Slug] {
		httpkit.WriteError(w, http.StatusBadRequest, "ese slug está reservado, elegí otro")
		return
	}

	course, err := h.Courses.Create(r.Context(), domain.Course{
		Title:       req.Title,
		Slug:        req.Slug,
		Description: req.Description,
	})
	if err != nil {
		if errors.Is(err, db.ErrConflict) {
			httpkit.WriteError(w, http.StatusConflict, "ya existe un curso con ese slug")
			return
		}
		log.Printf("admin: creando curso: %v", err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo crear el curso")
		return
	}

	httpkit.WriteJSON(w, http.StatusCreated, toCourseResponse(course, false))
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
		httpkit.WriteError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}

	if err := h.Courses.SetPublished(r.Context(), slug, req.Published); err != nil {
		if httpkit.HandleNotFound(w, err, "curso no encontrado") {
			return
		}
		log.Printf("admin: publicando curso %q: %v", slug, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo actualizar el curso")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Certificado ---

type setCertificateEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// SetCertificateEnabled prende o apaga la emisión automática de certificado
// al completar este curso al 100% (ver internal/certificate).
func (h *AdminHandler) SetCertificateEnabled(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var req setCertificateEnabledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpkit.WriteError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}

	if err := h.Courses.SetCertificateEnabled(r.Context(), slug, req.Enabled); err != nil {
		if httpkit.HandleNotFound(w, err, "curso no encontrado") {
			return
		}
		log.Printf("admin: actualizando certificate_enabled de curso %q: %v", slug, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo actualizar el curso")
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
// slug (ver coursepg.Repository.UpdateDetails) ni el estado de publicación — eso
// sigue siendo SetPublished, a propósito, para no mezclar "editar contenido"
// con "hacerlo visible".
func (h *AdminHandler) UpdateCourseDetails(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var req updateCourseDetailsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpkit.WriteError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	title, ok := httpkit.RequireNonEmpty(w, req.Title, "título")
	if !ok {
		return
	}
	req.Title = title

	if err := h.Courses.UpdateDetails(r.Context(), slug, req.Title, req.Description); err != nil {
		if httpkit.HandleNotFound(w, err, "curso no encontrado") {
			return
		}
		log.Printf("admin: actualizando curso %q: %v", slug, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo actualizar el curso")
		return
	}

	course, err := h.Courses.GetBySlugAny(r.Context(), slug)
	if err != nil {
		log.Printf("admin: releyendo curso %q tras actualizar: %v", slug, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "el curso se actualizó pero no se pudo confirmar")
		return
	}

	httpkit.WriteJSON(w, http.StatusOK, toAdminCourseResponse(course))
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
		if httpkit.HandleNotFound(w, err, "curso no encontrado") {
			return
		}
		log.Printf("admin: obteniendo curso %q para borrar: %v", slug, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo borrar el curso")
		return
	}

	lessons, err := h.Lessons.ListAllByCourse(r.Context(), course.ID)
	if err != nil {
		log.Printf("admin: listando lecciones del curso %d antes de borrar: %v", course.ID, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo borrar el curso")
		return
	}
	for _, lesson := range lessons {
		h.cleanupLessonPDFs(r.Context(), lesson.ID)
	}

	if err := h.Courses.Delete(r.Context(), slug); err != nil {
		if httpkit.HandleNotFound(w, err, "curso no encontrado") {
			return
		}
		log.Printf("admin: borrando curso %q: %v", slug, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo borrar el curso")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// cleanupLessonPDFs borra de R2 los PDFs de una lección, best-effort: si R2
// no está configurado o un borrado puntual falla, lo loguea y sigue — nunca
// bloquea un borrado en la base por un problema de storage. Copia
// deliberada de la misma lógica en lesson/infrastructure/http (ver el
// comentario de contentResponse más arriba sobre por qué no se comparte).
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
		if c.Type != lessondomain.ContentTypePDF || c.PDFURL == nil {
			continue
		}
		if err := h.R2.DeleteByURL(ctx, *c.PDFURL); err != nil {
			log.Printf("admin: borrando PDF de R2 (%s): %v", *c.PDFURL, err)
		}
	}
}
