// Package domain contiene la entidad Certificate y el puerto (Repository)
// que infrastructure/postgres implementa. No conoce SQL ni HTTP.
package domain

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound indica que no existe un certificado con el código dado.
var ErrNotFound = errors.New("certificate: not found")

// Certificate certifica que un usuario completó un curso. Code es el
// identificador público que viaja en la URL de verificación y en el QR: se
// genera con auth.GenerateToken (mismo crypto/rand que los tokens de un
// solo uso) pero, a diferencia de esos, se guarda sin hashear — el alumno
// necesita poder recuperar su certificado en cada login, y un hash SHA-256
// no es reversible.
//
// RecipientName y CourseTitle van desnormalizados a propósito: un
// certificado dice lo que decía cuando se emitió, no lo que el curso o el
// usuario se llaman hoy.
type Certificate struct {
	UserID        int64
	CourseID      int64
	Code          string
	RecipientName string
	CourseTitle   string
	IssuedAt      time.Time
}

// Repository es el puerto que expone infrastructure/postgres.
type Repository interface {
	// Issue inserta un certificado si no existe todavía uno para ese
	// usuario+curso (UNIQUE(user_id, course_id)) — idempotente: volver a
	// llamar tras ya haber emitido uno no crea un segundo.
	Issue(ctx context.Context, c Certificate) error
	// GetByCode busca un certificado por su código público. ErrNotFound si
	// no existe.
	GetByCode(ctx context.Context, code string) (Certificate, error)
	// CodesByUser devuelve, por curso, el código de certificado ya emitido
	// para ese usuario — solo incluye los cursos con certificado emitido.
	CodesByUser(ctx context.Context, userID int64) (map[int64]string, error)
}
