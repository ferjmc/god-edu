// Package domain contiene las entidades Course y CourseProgress y el
// puerto (Repository) que infrastructure/postgres implementa. No conoce
// SQL ni HTTP.
package domain

import (
	"context"
	"time"

	userdomain "github.com/ferjmc/god-edu/api/internal/user/domain"
)

// Course es un curso publicable de la plataforma. Un curso agrupa lecciones
// (ver internal/lesson) y solo se muestra a los usuarios cuando Published
// es true.
type Course struct {
	ID          int64
	Title       string
	Slug        string
	Description *string // nil si no se cargó descripción
	Published   bool
	// VisibleRoles restringe el detalle del curso a estos roles. Vacío
	// significa "sin restricción": visible para cualquier usuario logueado.
	// Es opt-in por curso, no opt-out (ver VisibleTo).
	VisibleRoles []userdomain.UserRole
	// CertificateEnabled indica si completar este curso al 100% emite un
	// certificado (ver internal/certificate). Configurable por el admin,
	// arranca en false.
	CertificateEnabled bool
	CreatedAt          time.Time
}

// VisibleTo indica si un usuario con el rol dado puede ver el detalle de
// este curso. ADMIN siempre puede, más allá de VisibleRoles. Un curso sin
// roles configurados es visible para cualquier rol.
func (c Course) VisibleTo(role userdomain.UserRole) bool {
	if role == userdomain.RoleAdmin {
		return true
	}
	if len(c.VisibleRoles) == 0 {
		return true
	}
	for _, r := range c.VisibleRoles {
		if r == role {
			return true
		}
	}
	return false
}

// CourseProgress es un curso visible para un usuario junto con su avance:
// cuántas lecciones tiene y cuántas ya completó. La arma
// Repository.ListVisibleWithProgress para la pantalla "mis cursos" — no
// lleva VisibleRoles (no hace falta una vez que el filtro de acceso ya se
// aplicó en la query).
type CourseProgress struct {
	ID                 int64
	Title              string
	Slug               string
	Description        *string
	CertificateEnabled bool
	TotalLessons       int
	CompletedLessons   int
	// NextLessonOrder es el order_index de la primera lección pendiente
	// (nil si el curso no tiene lecciones o ya están todas completadas) —
	// permite que "Continuar"/"Empezar" salten directo a la lección en vez
	// de la portada del curso.
	NextLessonOrder *int
}

// Repository es el puerto que expone infrastructure/postgres.
type Repository interface {
	// ListPublished devuelve los cursos publicados, más nuevos primero.
	ListPublished(ctx context.Context) ([]Course, error)
	// ListAll devuelve todos los cursos, publicados y en borrador, más
	// nuevos primero.
	ListAll(ctx context.Context) ([]Course, error)
	// GetBySlug busca un curso publicado por slug. Devuelve ErrNotFound
	// tanto si no existe como si existe pero todavía no está publicado —
	// desde afuera ambos casos son indistinguibles a propósito, para no
	// filtrar qué slugs existen en borrador.
	GetBySlug(ctx context.Context, slug string) (Course, error)
	// GetBySlugAny busca un curso por slug sin filtrar por published — la
	// usa el panel admin, que necesita poder ver y editar un curso mientras
	// todavía está en borrador.
	GetBySlugAny(ctx context.Context, slug string) (Course, error)
	// Create inserta un curso nuevo. Arranca siempre en borrador
	// (published = false) — publicarlo es un paso aparte (ver SetPublished).
	Create(ctx context.Context, c Course) (Course, error)
	// UpdateDetails actualiza título y descripción de un curso. El slug no
	// se puede tocar acá a propósito: cambiarlo rompería cualquier link ya
	// compartido a /cursos/{slug}.
	UpdateDetails(ctx context.Context, slug, title string, description *string) error
	// Delete borra un curso y, en cascada, sus lecciones, contenido,
	// inscripciones y progreso.
	Delete(ctx context.Context, slug string) error
	// SetPublished cambia el estado de publicación de un curso.
	SetPublished(ctx context.Context, slug string, published bool) error
	// SetCertificateEnabled prende o apaga la emisión de certificado para
	// un curso.
	SetCertificateEnabled(ctx context.Context, slug string, enabled bool) error
	// ListVisibleWithProgress devuelve, para un usuario y su rol, los
	// cursos publicados y visibles para ese rol junto con su progreso (ver
	// CourseProgress).
	ListVisibleWithProgress(ctx context.Context, userID int64, role userdomain.UserRole) ([]CourseProgress, error)
}
