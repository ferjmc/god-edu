// Package application orquesta los casos de uso de lecciones y su
// contenido sobre el puerto domain.Repository. No conoce SQL ni HTTP.
package application

import (
	"context"

	"github.com/ferjmc/god-edu/api/internal/lesson/domain"
)

// Service es el punto de entrada del dominio lesson para el resto de la
// API (hoy, infrastructure/http). Wrapper delgado sobre el
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

func (s *Service) RecordView(ctx context.Context, lessonID, userID int64) error {
	return s.repo.RecordView(ctx, lessonID, userID)
}

func (s *Service) ListActivityByUser(ctx context.Context, userID int64) ([]domain.UserLessonActivity, error) {
	return s.repo.ListActivityByUser(ctx, userID)
}

// CreateLesson crea una lección, opcionalmente dentro de la clase con
// order_index classOrder. La clase se busca siempre dentro de l.CourseID, así
// que una lección nunca puede quedar colgada de una clase de otro curso —
// si no existe ahí, devuelve db.ErrNotFound.
func (s *Service) CreateLesson(ctx context.Context, l domain.Lesson, classOrder *int) (domain.Lesson, error) {
	classID, err := s.resolveClassID(ctx, l.CourseID, classOrder)
	if err != nil {
		return domain.Lesson{}, err
	}
	l.ClassID = classID
	return s.repo.CreateLesson(ctx, l)
}

// SetLessonClass mueve una lección a la clase con order_index classOrder
// dentro de su mismo curso, o la deja sin clase si classOrder es nil.
func (s *Service) SetLessonClass(ctx context.Context, l domain.Lesson, classOrder *int) error {
	classID, err := s.resolveClassID(ctx, l.CourseID, classOrder)
	if err != nil {
		return err
	}
	return s.repo.SetLessonClass(ctx, l.ID, classID)
}

func (s *Service) resolveClassID(ctx context.Context, courseID int64, classOrder *int) (*int64, error) {
	if classOrder == nil {
		return nil, nil
	}
	c, err := s.repo.GetClassByCourseAndOrder(ctx, courseID, *classOrder)
	if err != nil {
		return nil, err
	}
	return &c.ID, nil
}

func (s *Service) ListClasses(ctx context.Context, courseID int64) ([]domain.Class, error) {
	return s.repo.ListClasses(ctx, courseID)
}

func (s *Service) GetClassByCourseAndOrder(ctx context.Context, courseID int64, order int) (domain.Class, error) {
	return s.repo.GetClassByCourseAndOrder(ctx, courseID, order)
}

func (s *Service) CreateClass(ctx context.Context, c domain.Class) (domain.Class, error) {
	return s.repo.CreateClass(ctx, c)
}

func (s *Service) UpdateClassTitle(ctx context.Context, classID int64, title string) error {
	return s.repo.UpdateClassTitle(ctx, classID, title)
}

func (s *Service) DeleteClass(ctx context.Context, classID int64) error {
	return s.repo.DeleteClass(ctx, classID)
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
