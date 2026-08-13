package handlers

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

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
