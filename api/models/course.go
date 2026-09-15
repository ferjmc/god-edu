package models

import "time"

// Course es un curso publicable de la plataforma. Un curso agrupa Lesson
// (ver lesson.go) y solo se muestra a los usuarios cuando Published es
// true.
type Course struct {
	ID          int64
	Title       string
	Slug        string
	Description *string // nil si no se cargó descripción
	Published   bool
	// VisibleRoles restringe el detalle del curso a estos roles. Vacío
	// significa "sin restricción": visible para cualquier usuario logueado.
	// Es opt-in por curso, no opt-out (ver VisibleTo).
	VisibleRoles []UserRole
	CreatedAt    time.Time
}

// VisibleTo indica si un usuario con el rol dado puede ver el detalle de
// este curso. ADMIN siempre puede, más allá de VisibleRoles. Un curso sin
// roles configurados es visible para cualquier rol.
func (c Course) VisibleTo(role UserRole) bool {
	if role == RoleAdmin {
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
// CourseRepo.ListVisibleWithProgress para la pantalla "mis cursos" — no
// lleva VisibleRoles (no hace falta una vez que el filtro de acceso ya se
// aplicó en la query).
type CourseProgress struct {
	ID               int64
	Title            string
	Slug             string
	Description      *string
	TotalLessons     int
	CompletedLessons int
	// NextLessonOrder es el order_index de la primera lección pendiente
	// (nil si el curso no tiene lecciones o ya están todas completadas) —
	// permite que "Continuar"/"Empezar" salten directo a la lección en vez
	// de la portada del curso.
	NextLessonOrder *int
}
