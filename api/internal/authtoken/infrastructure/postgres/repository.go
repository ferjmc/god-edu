// Package postgres implementa domain.Repository contra la tabla
// auth_tokens (links de verificación de email y de reset de password).
// Única capa del dominio authtoken que sabe que existe Postgres/pgx —
// mismo estilo (SQL crudo, sin ORM) que api/db.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ferjmc/god-edu/api/db"
	"github.com/ferjmc/god-edu/api/internal/authtoken/domain"
)

// Repository implementa domain.Repository. Los errores "no encontrado"
// siguen siendo db.ErrNotFound (no un sentinel propio) — mismo patrón que
// course/lesson/user.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create invalida los tokens sin usar del mismo propósito para ese usuario
// y crea uno nuevo, en una sola transacción: así nunca queda más de un
// link "vivo" al mismo tiempo (por ejemplo, si alguien pide reset de
// password dos veces, solo el último link sirve).
func (r *Repository) Create(ctx context.Context, userID int64, purpose domain.TokenPurpose, tokenHash string, ttl time.Duration) (domain.AuthToken, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.AuthToken{}, fmt.Errorf("authtoken: iniciando tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op si ya se hizo commit

	_, err = tx.Exec(ctx, `
		UPDATE auth_tokens SET used_at = now()
		WHERE user_id = $1 AND purpose = $2 AND used_at IS NULL
	`, userID, purpose)
	if err != nil {
		return domain.AuthToken{}, fmt.Errorf("authtoken: invalidando tokens previos: %w", err)
	}

	t := domain.AuthToken{
		UserID:    userID,
		Purpose:   purpose,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(ttl),
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO auth_tokens (user_id, token_hash, purpose, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, t.UserID, t.TokenHash, t.Purpose, t.ExpiresAt).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return domain.AuthToken{}, fmt.Errorf("authtoken: creando token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.AuthToken{}, fmt.Errorf("authtoken: commit: %w", err)
	}

	return t, nil
}

// GetValidByHash busca un token no usado y no expirado, del propósito dado.
// Si no hay match devuelve db.ErrNotFound sin distinguir si es porque no
// existe, ya se usó o expiró: a alguien intentando adivinar tokens no le
// conviene dar pistas de cuál es el motivo exacto.
func (r *Repository) GetValidByHash(ctx context.Context, tokenHash string, purpose domain.TokenPurpose) (domain.AuthToken, error) {
	const q = `
		SELECT id, user_id, token_hash, purpose, expires_at, used_at, created_at
		FROM auth_tokens
		WHERE token_hash = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > now()
	`
	var t domain.AuthToken
	err := r.pool.QueryRow(ctx, q, tokenHash, purpose).
		Scan(&t.ID, &t.UserID, &t.TokenHash, &t.Purpose, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AuthToken{}, db.ErrNotFound
		}
		return domain.AuthToken{}, fmt.Errorf("authtoken: leyendo token: %w", err)
	}
	return t, nil
}

// MarkUsed marca un token como consumido para que no se pueda reusar.
func (r *Repository) MarkUsed(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `UPDATE auth_tokens SET used_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("authtoken: marcando token usado: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return db.ErrNotFound
	}
	return nil
}
