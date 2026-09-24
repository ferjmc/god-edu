// Package domain contiene la entidad Enrollment y el puerto (Repository)
// que infrastructure/postgres implementa. No conoce SQL ni HTTP.
package domain

import (
	"context"
	"time"
)

// Enrollment registra que un usuario tomó un curso, y desde cuándo. No
// gatea acceso al contenido — el acceso sigue siendo por rol (ver
// handlers.resolveVisibleCourse). Es tracking, y la base para poder sumar
// precio/pasarela de pago en una etapa futura.
type Enrollment struct {
	UserID     int64
	CourseID   int64
	EnrolledAt time.Time
}

// Repository es el puerto que expone infrastructure/postgres.
type Repository interface {
	// Enroll inscribe al usuario en el curso. Idempotente: si ya estaba
	// inscripto, no hace nada (no es un error).
	Enroll(ctx context.Context, userID, courseID int64) error
	// IsEnrolled indica si el usuario ya está inscripto en el curso.
	IsEnrolled(ctx context.Context, userID, courseID int64) (bool, error)
	// EnrolledAtByUser devuelve, por curso, la fecha de inscripción de este
	// usuario — solo incluye los cursos en los que está inscripto.
	EnrolledAtByUser(ctx context.Context, userID int64) (map[int64]time.Time, error)
}
