package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ferjmc/god-edu/api/auth"
	"github.com/ferjmc/god-edu/api/db"
	"github.com/ferjmc/god-edu/api/models"
)

// CourseHandler agrupa los endpoints de cursos: List es público, Detail
// requiere sesión (ver newRouter en main.go).
type CourseHandler struct {
	Courses *db.CourseRepo
	Users   *db.UserRepo
}

// courseResponse es la representación pública de un curso. Solo lleva lo
// que necesita una tarjeta de listado; el detalle completo (lecciones,
// contenido) se resuelve en un endpoint aparte más adelante.
type courseResponse struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Slug        string  `json:"slug"`
	Description *string `json:"description"`
}

func toCourseResponse(c models.Course) courseResponse {
	return courseResponse{
		ID:          c.ID,
		Title:       c.Title,
		Slug:        c.Slug,
		Description: c.Description,
	}
}

// List devuelve los cursos publicados. Es público: no requiere sesión — el
// listado funciona como vidriera para usuarios anónimos, para invitarlos a
// registrarse. El detalle de cada curso sí requiere sesión (ver Detail).
func (h *CourseHandler) List(w http.ResponseWriter, r *http.Request) {
	courses, err := h.Courses.ListPublished(r.Context())
	if err != nil {
		log.Printf("courses: listando publicados: %v", err)
		writeError(w, http.StatusInternalServerError, "no se pudieron obtener los cursos")
		return
	}

	response := make([]courseResponse, len(courses))
	for i, c := range courses {
		response[i] = toCourseResponse(c)
	}

	writeJSON(w, http.StatusOK, response)
}

// Detail devuelve un curso por slug. Requiere sesión: un usuario anónimo no
// debe poder ver el contenido de un curso, solo su ficha en el listado.
// Además, si el curso restringe roles (course.VisibleRoles), el usuario
// logueado tiene que tener uno de esos roles — devuelve 403 si no (ver
// resolveVisibleCourse, que comparte con LessonHandler).
func (h *CourseHandler) Detail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	course, err := resolveVisibleCourse(r.Context(), h.Courses, h.Users, slug)
	if err != nil {
		writeCourseAccessError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toCourseResponse(course))
}

type courseProgressResponse struct {
	ID               int64   `json:"id"`
	Title            string  `json:"title"`
	Slug             string  `json:"slug"`
	Description      *string `json:"description"`
	TotalLessons     int     `json:"totalLessons"`
	CompletedLessons int     `json:"completedLessons"`
	ProgressPercent  int     `json:"progressPercent"` // 0 si totalLessons == 0
	// NextLessonOrder es el order_index de la primera lección pendiente —
	// nil si el curso no tiene lecciones o ya las completó todas, en cuyo
	// caso el frontend cae de vuelta a la portada del curso.
	NextLessonOrder *int `json:"nextLessonOrder,omitempty"`
}

func toCourseProgressResponse(c models.CourseProgress) courseProgressResponse {
	percent := 0
	if c.TotalLessons > 0 {
		percent = int(float64(c.CompletedLessons) / float64(c.TotalLessons) * 100)
	}
	return courseProgressResponse{
		ID: c.ID, Title: c.Title, Slug: c.Slug, Description: c.Description,
		TotalLessons: c.TotalLessons, CompletedLessons: c.CompletedLessons,
		ProgressPercent: percent, NextLessonOrder: c.NextLessonOrder,
	}
}

// Mine devuelve, para el usuario autenticado, todos los cursos visibles
// para su rol junto con su progreso — no hay inscripción explícita: "mis
// cursos" es lo mismo que "los cursos a los que tengo acceso" (ver
// Course.VisibleTo). La tabla enrollments queda sin usar, a propósito.
func (h *CourseHandler) Mine(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context()) // RequireAuth ya lo garantiza

	user, err := h.Users.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// El JWT es válido pero el usuario ya no existe (cuenta borrada
			// después de emitido el token) — la sesión quedó huérfana.
			writeError(w, http.StatusUnauthorized, "sesión inválida")
			return
		}
		log.Printf("courses: buscando usuario %d: %v", userID, err)
		writeError(w, http.StatusInternalServerError, "no se pudieron obtener tus cursos")
		return
	}

	courses, err := h.Courses.ListVisibleWithProgress(r.Context(), userID, user.Role)
	if err != nil {
		log.Printf("courses: listando cursos con progreso del usuario %d: %v", userID, err)
		writeError(w, http.StatusInternalServerError, "no se pudieron obtener tus cursos")
		return
	}

	response := make([]courseProgressResponse, len(courses))
	for i, c := range courses {
		response[i] = toCourseProgressResponse(c)
	}
	writeJSON(w, http.StatusOK, response)
}
