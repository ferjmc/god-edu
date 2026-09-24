package handlers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ferjmc/god-edu/api/auth"
	"github.com/ferjmc/god-edu/api/db"
	certapp "github.com/ferjmc/god-edu/api/internal/certificate/application"
	enrollapp "github.com/ferjmc/god-edu/api/internal/enrollment/application"
	"github.com/ferjmc/god-edu/api/models"
)

// CourseHandler agrupa los endpoints de cursos: List es público, el resto
// requiere sesión (ver newRouter en main.go).
type CourseHandler struct {
	Courses      *db.CourseRepo
	Users        *db.UserRepo
	Enrollments  *enrollapp.Service
	Certificates *certapp.Service
}

// courseResponse es la representación pública de un curso. Solo lleva lo
// que necesita una tarjeta de listado o el detalle; el listado de lecciones
// se resuelve en un endpoint aparte (ver LessonHandler).
type courseResponse struct {
	ID                 int64   `json:"id"`
	Title              string  `json:"title"`
	Slug               string  `json:"slug"`
	Description        *string `json:"description"`
	CertificateEnabled bool    `json:"certificateEnabled"`
	// Enrolled indica si el usuario autenticado ya se inscribió en este
	// curso (ver POST .../enroll). No afecta si puede ver esta respuesta —
	// eso lo decide el rol (ver resolveVisibleCourse) — solo si el frontend
	// debe mostrar el panel completo de lecciones o la pantalla de resumen
	// con el botón "Inscribirme". No va en List: ahí no hay un usuario en
	// particular para el que resolverlo sin una consulta por curso.
	Enrolled bool `json:"enrolled"`
}

func toCourseResponse(c models.Course, enrolled bool) courseResponse {
	return courseResponse{
		ID:                 c.ID,
		Title:              c.Title,
		Slug:               c.Slug,
		Description:        c.Description,
		CertificateEnabled: c.CertificateEnabled,
		Enrolled:           enrolled,
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

	// List es la vidriera pública (sin sesión): no hay un usuario para el
	// que resolver "enrolled", así que siempre va en false. El frontend no
	// lo necesita acá — solo importa en el detalle (ver Detail).
	response := make([]courseResponse, len(courses))
	for i, c := range courses {
		response[i] = toCourseResponse(c, false)
	}

	writeJSON(w, http.StatusOK, response)
}

// Detail devuelve un curso por slug. Requiere sesión: un usuario anónimo no
// debe poder ver el contenido de un curso, solo su ficha en el listado.
// Además, si el curso restringe roles (course.VisibleRoles), el usuario
// logueado tiene que tener uno de esos roles — devuelve 403 si no (ver
// resolveVisibleCourse, que comparte con LessonHandler). No inscribe
// automáticamente: la inscripción es una acción explícita del alumno (ver
// Enroll) — el frontend usa el campo Enrolled de la respuesta para decidir
// si mostrar el resumen con el botón "Inscribirme" o el panel completo.
func (h *CourseHandler) Detail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	course, err := resolveVisibleCourse(r.Context(), h.Courses, h.Users, slug)
	if err != nil {
		writeCourseAccessError(w, err)
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())
	enrolled, err := h.Enrollments.IsEnrolled(r.Context(), userID, course.ID)
	if err != nil {
		log.Printf("courses: consultando inscripción (usuario %d, curso %d): %v", userID, course.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudo obtener el curso")
		return
	}

	writeJSON(w, http.StatusOK, toCourseResponse(course, enrolled))
}

// Enroll inscribe al usuario autenticado en el curso. Vuelve a correr el
// mismo chequeo de acceso por rol que Detail (resolveVisibleCourse) — nunca
// hay que confiar en que el botón "Inscribirme" del frontend fue el único
// gate. Idempotente: llamarlo dos veces no crea una segunda inscripción
// (ver enrollment/infrastructure/postgres.Repository.Enroll). No cambia
// nada del control de acceso: seguir sin estar inscripto no bloquea nada
// que el rol ya permitiera ver — es tracking, no un gate de seguridad.
func (h *CourseHandler) Enroll(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	course, err := resolveVisibleCourse(r.Context(), h.Courses, h.Users, slug)
	if err != nil {
		writeCourseAccessError(w, err)
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())
	if err := h.Enrollments.Enroll(r.Context(), userID, course.ID); err != nil {
		log.Printf("courses: inscribiendo usuario %d en curso %d: %v", userID, course.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudo completar la inscripción")
		return
	}

	writeJSON(w, http.StatusOK, toCourseResponse(course, true))
}

type courseProgressResponse struct {
	ID                 int64   `json:"id"`
	Title              string  `json:"title"`
	Slug               string  `json:"slug"`
	Description        *string `json:"description"`
	CertificateEnabled bool    `json:"certificateEnabled"`
	TotalLessons       int     `json:"totalLessons"`
	CompletedLessons   int     `json:"completedLessons"`
	ProgressPercent    int     `json:"progressPercent"` // 0 si totalLessons == 0
	// NextLessonOrder es el order_index de la primera lección pendiente —
	// nil si el curso no tiene lecciones o ya las completó todas, en cuyo
	// caso el frontend cae de vuelta a la portada del curso.
	NextLessonOrder *int `json:"nextLessonOrder,omitempty"`
	// EnrolledAt y CertificateCode se completan aparte (ver Mine): no salen
	// de ListVisibleWithProgress, que no sabe nada de enrollment/certificate.
	// Ambos van con omitempty: la mayoría de los cursos en este listado no
	// va a tener ninguno de los dos.
	EnrolledAt      *time.Time `json:"enrolledAt,omitempty"`
	CertificateCode *string    `json:"certificateCode,omitempty"`
}

func toCourseProgressResponse(c models.CourseProgress) courseProgressResponse {
	percent := 0
	if c.TotalLessons > 0 {
		percent = int(float64(c.CompletedLessons) / float64(c.TotalLessons) * 100)
	}
	return courseProgressResponse{
		ID: c.ID, Title: c.Title, Slug: c.Slug, Description: c.Description,
		CertificateEnabled: c.CertificateEnabled,
		TotalLessons:       c.TotalLessons, CompletedLessons: c.CompletedLessons,
		ProgressPercent: percent, NextLessonOrder: c.NextLessonOrder,
	}
}

// Mine devuelve, para el usuario autenticado, todos los cursos visibles
// para su rol junto con su progreso (Course.VisibleTo sigue siendo el
// control de acceso real). Además completa, por curso, la fecha de
// inscripción y el código de certificado si corresponden — enrollment y
// certificate son dominios aparte (ver internal/), así que se resuelven acá
// con dos consultas más y se mergean en memoria, no en el JOIN de
// ListVisibleWithProgress.
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

	enrolledAt, err := h.Enrollments.EnrolledAtByUser(r.Context(), userID)
	if err != nil {
		log.Printf("courses: obteniendo inscripciones del usuario %d: %v", userID, err)
		writeError(w, http.StatusInternalServerError, "no se pudieron obtener tus cursos")
		return
	}
	certCodes, err := h.Certificates.CodesByUser(r.Context(), userID)
	if err != nil {
		log.Printf("courses: obteniendo certificados del usuario %d: %v", userID, err)
		writeError(w, http.StatusInternalServerError, "no se pudieron obtener tus cursos")
		return
	}

	response := make([]courseProgressResponse, len(courses))
	for i, c := range courses {
		cr := toCourseProgressResponse(c)
		if at, ok := enrolledAt[c.ID]; ok {
			cr.EnrolledAt = &at
		}
		if code, ok := certCodes[c.ID]; ok {
			cr.CertificateCode = &code
		}
		response[i] = cr
	}
	writeJSON(w, http.StatusOK, response)
}
