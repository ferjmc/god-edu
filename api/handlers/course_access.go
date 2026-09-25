package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/ferjmc/god-edu/api/auth"
	"github.com/ferjmc/god-edu/api/db"
	courseapp "github.com/ferjmc/god-edu/api/internal/course/application"
	coursedomain "github.com/ferjmc/god-edu/api/internal/course/domain"
	userapp "github.com/ferjmc/god-edu/api/internal/user/application"
)

// errForbidden indica que el usuario está autenticado pero no tiene un rol
// habilitado para ver este curso (ver coursedomain.Course.VisibleTo).
var errForbidden = errors.New("acceso no autorizado a este curso")

// errUnauthenticated indica que no había un usuario válido en el contexto.
// No debería pasar nunca en la práctica — todo endpoint que llama a
// resolveVisibleCourse está detrás de auth.RequireAuth — pero el handler
// lo mapea a 401 igual en vez de asumir.
var errUnauthenticated = errors.New("no autenticado")

// resolveVisibleCourse busca un curso publicado por slug y confirma que el
// usuario autenticado en el contexto tiene un rol habilitado para verlo.
// La comparten CourseHandler.Detail y LessonHandler (List y Detail):
// cualquier endpoint que exponga contenido de un curso — no solo su ficha
// de listado — pasa por acá antes de tocar la base, para no repetir el
// mismo chequeo de rol en cada uno.
func resolveVisibleCourse(ctx context.Context, courses *courseapp.Service, users *userapp.Service, slug string) (coursedomain.Course, error) {
	course, err := courses.GetBySlug(ctx, slug)
	if err != nil {
		return coursedomain.Course{}, err // db.ErrNotFound se propaga tal cual
	}

	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return coursedomain.Course{}, errUnauthenticated
	}
	user, err := users.GetByID(ctx, userID)
	if err != nil {
		return coursedomain.Course{}, errUnauthenticated
	}

	if !course.VisibleTo(user.Role) {
		return coursedomain.Course{}, errForbidden
	}

	return course, nil
}

// writeCourseAccessError traduce un error de resolveVisibleCourse al
// status HTTP correspondiente.
func writeCourseAccessError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, db.ErrNotFound):
		writeError(w, http.StatusNotFound, "curso no encontrado")
	case errors.Is(err, errForbidden):
		writeError(w, http.StatusForbidden, "no tenés acceso a este curso")
	case errors.Is(err, errUnauthenticated):
		writeError(w, http.StatusUnauthorized, "no autenticado")
	default:
		log.Printf("handlers: resolviendo acceso a curso: %v", err)
		writeError(w, http.StatusInternalServerError, "no se pudo obtener el curso")
	}
}
