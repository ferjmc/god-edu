// Package application orquesta la emisión y verificación de certificados
// sobre el puerto domain.Repository. No conoce SQL ni HTTP.
package application

import (
	"context"

	"github.com/ferjmc/god-edu/api/auth"
	"github.com/ferjmc/god-edu/api/internal/certificate/domain"
)

// Service es el punto de entrada del dominio certificate para el resto de
// la API (hoy, los handlers en api/handlers).
type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

// IssueInput son los datos necesarios para emitir un certificado.
type IssueInput struct {
	UserID        int64
	CourseID      int64
	RecipientName string
	CourseTitle   string
}

// IssueIfEligible genera un código nuevo y lo persiste. El nombre refleja
// que la elegibilidad (100% del curso completado y course.CertificateEnabled)
// la decide el caller (LessonHandler.MarkComplete) antes de llamar acá: esa
// información vive en los dominios Course/Lesson, no en este — evita que
// certificate dependa de courseapp/lessonapp solo para repetir un
// chequeo que el caller ya hizo. Issue en sí es idempotente (constraint
// UNIQUE(user_id, course_id) del lado del repositorio): llamar dos veces
// para el mismo usuario+curso no emite un segundo certificado.
func (s *Service) IssueIfEligible(ctx context.Context, in IssueInput) error {
	code, _, err := auth.GenerateToken()
	if err != nil {
		return err
	}

	return s.repo.Issue(ctx, domain.Certificate{
		UserID:        in.UserID,
		CourseID:      in.CourseID,
		Code:          code,
		RecipientName: in.RecipientName,
		CourseTitle:   in.CourseTitle,
	})
}

// VerifyByCode busca un certificado por su código público.
func (s *Service) VerifyByCode(ctx context.Context, code string) (domain.Certificate, error) {
	return s.repo.GetByCode(ctx, code)
}

// CodesByUser devuelve, por curso, el código de certificado ya emitido para
// ese usuario.
func (s *Service) CodesByUser(ctx context.Context, userID int64) (map[int64]string, error) {
	return s.repo.CodesByUser(ctx, userID)
}
