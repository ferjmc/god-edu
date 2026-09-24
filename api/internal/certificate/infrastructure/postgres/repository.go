// Package postgres implementa domain.Repository contra la tabla
// certificates. Única capa del dominio certificate que sabe que existe
// Postgres/pgx — mismo estilo (SQL crudo, sin ORM) que api/db.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ferjmc/god-edu/api/internal/certificate/domain"
)

// Repository implementa domain.Repository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Issue inserta un certificado si no existe todavía uno para ese
// usuario+curso (UNIQUE(user_id, course_id) — ver migrations/000005) — no
// falla ni pisa el código existente si ya había uno.
func (r *Repository) Issue(ctx context.Context, c domain.Certificate) error {
	const q = `
		INSERT INTO certificates (user_id, course_id, code, recipient_name, course_title)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, course_id) DO NOTHING
	`
	if _, err := r.pool.Exec(ctx, q, c.UserID, c.CourseID, c.Code, c.RecipientName, c.CourseTitle); err != nil {
		return fmt.Errorf("certificate: emitiendo certificado (usuario %d, curso %d): %w", c.UserID, c.CourseID, err)
	}
	return nil
}

func (r *Repository) GetByCode(ctx context.Context, code string) (domain.Certificate, error) {
	const q = `
		SELECT user_id, course_id, code, recipient_name, course_title, issued_at
		FROM certificates
		WHERE code = $1
	`
	var c domain.Certificate
	err := r.pool.QueryRow(ctx, q, code).
		Scan(&c.UserID, &c.CourseID, &c.Code, &c.RecipientName, &c.CourseTitle, &c.IssuedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Certificate{}, domain.ErrNotFound
		}
		return domain.Certificate{}, fmt.Errorf("certificate: buscando certificado %q: %w", code, err)
	}
	return c, nil
}

func (r *Repository) CodesByUser(ctx context.Context, userID int64) (map[int64]string, error) {
	const q = `SELECT course_id, code FROM certificates WHERE user_id = $1`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("certificate: listando certificados del usuario %d: %w", userID, err)
	}
	defer rows.Close()

	result := make(map[int64]string)
	for rows.Next() {
		var courseID int64
		var code string
		if err := rows.Scan(&courseID, &code); err != nil {
			return nil, fmt.Errorf("certificate: leyendo certificado del usuario %d: %w", userID, err)
		}
		result[courseID] = code
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("certificate: iterando certificados del usuario %d: %w", userID, err)
	}
	return result, nil
}
