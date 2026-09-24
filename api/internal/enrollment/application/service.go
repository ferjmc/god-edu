// Package application orquesta los casos de uso de inscripción sobre el
// puerto domain.Repository. No conoce SQL ni HTTP.
package application

import (
	"context"
	"time"

	"github.com/ferjmc/god-edu/api/internal/enrollment/domain"
)

// Service es el punto de entrada del dominio enrollment para el resto de
// la API (hoy, los handlers en api/handlers).
type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

// Enroll inscribe al usuario en el curso. Idempotente.
func (s *Service) Enroll(ctx context.Context, userID, courseID int64) error {
	return s.repo.Enroll(ctx, userID, courseID)
}

// IsEnrolled indica si el usuario ya está inscripto en el curso.
func (s *Service) IsEnrolled(ctx context.Context, userID, courseID int64) (bool, error) {
	return s.repo.IsEnrolled(ctx, userID, courseID)
}

// EnrolledAtByUser devuelve, por curso, la fecha de inscripción del usuario.
func (s *Service) EnrolledAtByUser(ctx context.Context, userID int64) (map[int64]time.Time, error) {
	return s.repo.EnrolledAtByUser(ctx, userID)
}
