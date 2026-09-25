// Package application orquesta los casos de uso de lecciones y su
// contenido sobre el puerto domain.Repository. No conoce SQL ni HTTP.
package application

import (
	"context"

	"github.com/ferjmc/god-edu/api/internal/lesson/domain"
)

// Service es el punto de entrada del dominio lesson para el resto de la
// API (hoy, los handlers en api/handlers). Wrapper delgado sobre el
// puerto.
type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListByCourse(ctx context.Context, courseID, userID int64) ([]domain.LessonSummary, error) {
	return s.repo.ListByCourse(ctx, courseID, userID)
}

func (s *Service) ListAllByCourse(ctx context.Context, courseID int64) ([]domain.Lesson, error) {
	return s.repo.ListAllByCourse(ctx, courseID)
}

func (s *Service) GetByCourseAndOrder(ctx context.Context, courseID int64, order int) (domain.Lesson, error) {
	return s.repo.GetByCourseAndOrder(ctx, courseID, order)
}

func (s *Service) UpdateTitle(ctx context.Context, lessonID int64, title string) error {
	return s.repo.UpdateTitle(ctx, lessonID, title)
}

func (s *Service) DeleteLesson(ctx context.Context, lessonID int64) error {
	return s.repo.DeleteLesson(ctx, lessonID)
}

func (s *Service) IsCompleted(ctx context.Context, lessonID, userID int64) (bool, error) {
	return s.repo.IsCompleted(ctx, lessonID, userID)
}

func (s *Service) MarkCompleted(ctx context.Context, lessonID, userID int64) error {
	return s.repo.MarkCompleted(ctx, lessonID, userID)
}

func (s *Service) CreateLesson(ctx context.Context, l domain.Lesson) (domain.Lesson, error) {
	return s.repo.CreateLesson(ctx, l)
}

func (s *Service) CreateContent(ctx context.Context, c domain.LessonContent) (domain.LessonContent, error) {
	return s.repo.CreateContent(ctx, c)
}

func (s *Service) GetContentByLessonAndOrder(ctx context.Context, lessonID int64, order int) (domain.LessonContent, error) {
	return s.repo.GetContentByLessonAndOrder(ctx, lessonID, order)
}

func (s *Service) UpdateContent(ctx context.Context, id int64, title string, youtubeURL, body *string) error {
	return s.repo.UpdateContent(ctx, id, title, youtubeURL, body)
}

func (s *Service) DeleteContent(ctx context.Context, id int64) error {
	return s.repo.DeleteContent(ctx, id)
}

func (s *Service) NextContentOrder(ctx context.Context, lessonID int64) (int, error) {
	return s.repo.NextContentOrder(ctx, lessonID)
}

func (s *Service) ListContent(ctx context.Context, lessonID int64) ([]domain.LessonContent, error) {
	return s.repo.ListContent(ctx, lessonID)
}
