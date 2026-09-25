package application

import (
	"context"
	"errors"
	"testing"

	"github.com/ferjmc/god-edu/api/internal/certificate/domain"
)

// fakeRepo implementa domain.Repository en memoria, indexado por código —
// Issue respeta la misma semántica que la constraint real
// UNIQUE(user_id, course_id): si ya hay un certificado para ese par, no lo
// pisa (mismo comportamiento que ON CONFLICT DO NOTHING en Postgres).
type fakeRepo struct {
	byCode map[string]domain.Certificate
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byCode: make(map[string]domain.Certificate)}
}

func (f *fakeRepo) Issue(ctx context.Context, c domain.Certificate) error {
	for _, existing := range f.byCode {
		if existing.UserID == c.UserID && existing.CourseID == c.CourseID {
			return nil
		}
	}
	f.byCode[c.Code] = c
	return nil
}

func (f *fakeRepo) GetByCode(ctx context.Context, code string) (domain.Certificate, error) {
	c, ok := f.byCode[code]
	if !ok {
		return domain.Certificate{}, domain.ErrNotFound
	}
	return c, nil
}

func (f *fakeRepo) CodesByUser(ctx context.Context, userID int64) (map[int64]string, error) {
	result := make(map[int64]string)
	for code, c := range f.byCode {
		if c.UserID == userID {
			result[c.CourseID] = code
		}
	}
	return result, nil
}

func TestService_IssueIfEligible(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := NewService(repo)

	in := IssueInput{UserID: 1, CourseID: 10, RecipientName: "Ada Lovelace", CourseTitle: "Curso de prueba"}
	if err := svc.IssueIfEligible(ctx, in); err != nil {
		t.Fatalf("no esperaba error, recibí %v", err)
	}

	codes, err := svc.CodesByUser(ctx, 1)
	if err != nil {
		t.Fatalf("CodesByUser: no esperaba error, recibí %v", err)
	}
	code, ok := codes[10]
	if !ok || code == "" {
		t.Fatalf("esperaba un código no vacío para el curso 10, recibí %q (presente=%v)", code, ok)
	}

	cert, err := svc.VerifyByCode(ctx, code)
	if err != nil {
		t.Fatalf("VerifyByCode: no esperaba error, recibí %v", err)
	}
	if cert.RecipientName != in.RecipientName || cert.CourseTitle != in.CourseTitle {
		t.Fatalf("certificado con datos incorrectos: %+v", cert)
	}

	// Idempotencia: volver a emitir para el mismo usuario+curso no genera
	// un segundo código.
	if err := svc.IssueIfEligible(ctx, in); err != nil {
		t.Fatalf("segunda llamada: no esperaba error, recibí %v", err)
	}
	codesAgain, err := svc.CodesByUser(ctx, 1)
	if err != nil {
		t.Fatalf("CodesByUser (2da vez): no esperaba error, recibí %v", err)
	}
	if codesAgain[10] != code {
		t.Fatalf("esperaba el mismo código tras reintentar: antes=%q ahora=%q", code, codesAgain[10])
	}
}

func TestService_VerifyByCode_NotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	_, err := svc.VerifyByCode(context.Background(), "codigo-inexistente")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("esperaba domain.ErrNotFound, recibí %v", err)
	}
}

func TestService_CodesByUser(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := NewService(repo)

	if err := svc.IssueIfEligible(ctx, IssueInput{UserID: 1, CourseID: 10, RecipientName: "A", CourseTitle: "X"}); err != nil {
		t.Fatalf("no esperaba error, recibí %v", err)
	}
	if err := svc.IssueIfEligible(ctx, IssueInput{UserID: 1, CourseID: 20, RecipientName: "A", CourseTitle: "Y"}); err != nil {
		t.Fatalf("no esperaba error, recibí %v", err)
	}
	if err := svc.IssueIfEligible(ctx, IssueInput{UserID: 2, CourseID: 10, RecipientName: "B", CourseTitle: "X"}); err != nil {
		t.Fatalf("no esperaba error, recibí %v", err)
	}

	codes, err := svc.CodesByUser(ctx, 1)
	if err != nil {
		t.Fatalf("no esperaba error, recibí %v", err)
	}
	if len(codes) != 2 || codes[10] == "" || codes[20] == "" {
		t.Fatalf("esperaba códigos para los cursos 10 y 20 del usuario 1, recibí %v", codes)
	}
}
