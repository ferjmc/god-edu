package http

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/ferjmc/god-edu/api/auth"
	"github.com/ferjmc/god-edu/api/db"
	"github.com/ferjmc/god-edu/api/httpkit"
	"github.com/ferjmc/god-edu/api/internal/course/application"
	"github.com/ferjmc/god-edu/api/internal/course/domain"
	userapp "github.com/ferjmc/god-edu/api/internal/user/application"
)

// errForbidden indica que el usuario está autenticado pero no tiene un rol
// habilitado para ver este curso (ver domain.Course.VisibleTo). No exportado
// a propósito: nada fuera de WriteCourseAccessError necesita distinguirlo —
// un caller externo debe tratar el error de ResolveVisibleCourse como
// opaco y dejar que WriteCourseAccessError decida el status HTTP.
var errForbidden = errors.New("acceso no autorizado a este curso")

// errUnauthenticated indica que no había un usuario válido en el contexto.
// No debería pasar nunca en la práctica — todo endpoint que llama a
// ResolveVisibleCourse está detrás de auth.RequireAuth — pero el handler
// lo mapea a 401 igual en vez de asumir.
var errUnauthenticated = errors.New("no autenticado")

// ResolveVisibleCourse busca un curso publicado por slug y confirma que el
// usuario autenticado en el contexto tiene un rol habilitado para verlo. La
// comparten CourseHandler.Detail acá y, cruzando a otro dominio,
// lesson/infrastructure/http.LessonHandler (List y Detail): cualquier
// endpoint que exponga contenido de un curso — no solo su ficha de listado
// — pasa por acá antes de tocar la base, para no repetir el mismo chequeo
// de rol en cada uno.
func ResolveVisibleCourse(ctx context.Context, courses *application.Service, users *userapp.Service, slug string) (domain.Course, error) {
	course, err := courses.GetBySlug(ctx, slug)
	if err != nil {
		return domain.Course{}, err // db.ErrNotFound se propaga tal cual
	}

	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return domain.Course{}, errUnauthenticated
	}
	user, err := users.GetByID(ctx, userID)
	if err != nil {
		return domain.Course{}, errUnauthenticated
	}

	if !course.VisibleTo(user.Role) {
		return domain.Course{}, errForbidden
	}

	return course, nil
}

// WriteCourseAccessError traduce un error de ResolveVisibleCourse al status
// HTTP correspondiente.
func WriteCourseAccessError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, db.ErrNotFound):
		httpkit.WriteError(w, http.StatusNotFound, "curso no encontrado")
	case errors.Is(err, errForbidden):
		httpkit.WriteError(w, http.StatusForbidden, "no tenés acceso a este curso")
	case errors.Is(err, errUnauthenticated):
		httpkit.WriteError(w, http.StatusUnauthorized, "no autenticado")
	default:
		log.Printf("course/http: resolviendo acceso a curso: %v", err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo obtener el curso")
	}
}
