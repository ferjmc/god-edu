// Package postgres implementa domain.Repository contra la tabla
// enrollments. Única capa del dominio enrollment que sabe que existe
// Postgres/pgx — mismo estilo (SQL crudo, sin ORM) que api/db.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository implementa domain.Repository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Enroll inserta la inscripción si no existe (ON CONFLICT DO NOTHING sobre
// UNIQUE(user_id, course_id) — ver migrations/000001) — llamar dos veces
// para el mismo usuario+curso no crea una segunda fila.
func (r *Repository) Enroll(ctx context.Context, userID, courseID int64) error {
	const q = `
		INSERT INTO enrollments (user_id, course_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, course_id) DO NOTHING
	`
	if _, err := r.pool.Exec(ctx, q, userID, courseID); err != nil {
		return fmt.Errorf("enrollment: inscribiendo usuario %d en curso %d: %w", userID, courseID, err)
	}
	return nil
}

func (r *Repository) IsEnrolled(ctx context.Context, userID, courseID int64) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM enrollments WHERE user_id = $1 AND course_id = $2)`
	var enrolled bool
	if err := r.pool.QueryRow(ctx, q, userID, courseID).Scan(&enrolled); err != nil {
		return false, fmt.Errorf("enrollment: consultando inscripción (usuario %d, curso %d): %w", userID, courseID, err)
	}
	return enrolled, nil
}

func (r *Repository) EnrolledAtByUser(ctx context.Context, userID int64) (map[int64]time.Time, error) {
	const q = `SELECT course_id, enrolled_at FROM enrollments WHERE user_id = $1`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("enrollment: listando inscripciones del usuario %d: %w", userID, err)
	}
	defer rows.Close()

	result := make(map[int64]time.Time)
	for rows.Next() {
		var courseID int64
		var enrolledAt time.Time
		if err := rows.Scan(&courseID, &enrolledAt); err != nil {
			return nil, fmt.Errorf("enrollment: leyendo inscripción del usuario %d: %w", userID, err)
		}
		result[courseID] = enrolledAt
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("enrollment: iterando inscripciones del usuario %d: %w", userID, err)
	}
	return result, nil
}
