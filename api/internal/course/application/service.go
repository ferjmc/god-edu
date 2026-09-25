// Package application orquesta los casos de uso de cursos sobre el puerto
// domain.Repository. No conoce SQL ni HTTP.
package application

import (
	"context"

	"github.com/ferjmc/god-edu/api/internal/course/domain"
	userdomain "github.com/ferjmc/god-edu/api/internal/user/domain"
)

// Service es el punto de entrada del dominio course para el resto de la
// API (hoy, los handlers en api/handlers). Wrapper delgado sobre el
// puerto — la validación de slug (formato, reservados) sigue viviendo en
// el handler admin, no acá, para no ampliar el alcance de esta migración.
type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListPublished(ctx context.Context) ([]domain.Course, error) {
	return s.repo.ListPublished(ctx)
}

func (s *Service) ListAll(ctx context.Context) ([]domain.Course, error) {
	return s.repo.ListAll(ctx)
}

func (s *Service) GetBySlug(ctx context.Context, slug string) (domain.Course, error) {
	return s.repo.GetBySlug(ctx, slug)
}

func (s *Service) GetBySlugAny(ctx context.Context, slug string) (domain.Course, error) {
	return s.repo.GetBySlugAny(ctx, slug)
}

func (s *Service) Create(ctx context.Context, c domain.Course) (domain.Course, error) {
	return s.repo.Create(ctx, c)
}

func (s *Service) UpdateDetails(ctx context.Context, slug, title string, description *string) error {
	return s.repo.UpdateDetails(ctx, slug, title, description)
}

func (s *Service) Delete(ctx context.Context, slug string) error {
	return s.repo.Delete(ctx, slug)
}

func (s *Service) SetPublished(ctx context.Context, slug string, published bool) error {
	return s.repo.SetPublished(ctx, slug, published)
}

func (s *Service) SetCertificateEnabled(ctx context.Context, slug string, enabled bool) error {
	return s.repo.SetCertificateEnabled(ctx, slug, enabled)
}

func (s *Service) ListVisibleWithProgress(ctx context.Context, userID int64, role userdomain.UserRole) ([]domain.CourseProgress, error) {
	return s.repo.ListVisibleWithProgress(ctx, userID, role)
}
