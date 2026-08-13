package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ferjmc/god-edu/api/models"
)

// uniqueViolation es el código de error de Postgres para restricciones
// UNIQUE (ver https://www.postgresql.org/docs/current/errcodes-appendix.html).
const uniqueViolation = "23505"

// UserRepo implementa el acceso a la tabla users.
type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// Create inserta un usuario nuevo y devuelve el registro con id/created_at
// completados por la base.
func (r *UserRepo) Create(ctx context.Context, u models.User) (models.User, error) {
	const q = `
		INSERT INTO users (email, password_hash, name, auth_provider, email_verified, role)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(ctx, q, u.Email, u.PasswordHash, u.Name, u.AuthProvider, u.EmailVerified, u.Role).
		Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return models.User{}, ErrConflict
		}
		return models.User{}, fmt.Errorf("db: creando usuario: %w", err)
	}

	return u, nil
}

// GetByEmail busca un usuario por email. Devuelve ErrNotFound si no existe.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (models.User, error) {
	const q = `
		SELECT id, email, password_hash, name, auth_provider, email_verified, role, created_at
		FROM users
		WHERE email = $1
	`
	return r.scanOne(r.pool.QueryRow(ctx, q, email))
}

// GetByID busca un usuario por id. Devuelve ErrNotFound si no existe.
func (r *UserRepo) GetByID(ctx context.Context, id int64) (models.User, error) {
	const q = `
		SELECT id, email, password_hash, name, auth_provider, email_verified, role, created_at
		FROM users
		WHERE id = $1
	`
	return r.scanOne(r.pool.QueryRow(ctx, q, id))
}

// MarkEmailVerified pone email_verified = true para el usuario dado.
func (r *UserRepo) MarkEmailVerified(ctx context.Context, id int64) error {
	const q = `UPDATE users SET email_verified = true WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("db: marcando email verificado: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdatePasswordHash reemplaza el password_hash de un usuario (flujo de
// reset de password).
func (r *UserRepo) UpdatePasswordHash(ctx context.Context, id int64, passwordHash string) error {
	const q = `UPDATE users SET password_hash = $1 WHERE id = $2`
	tag, err := r.pool.Exec(ctx, q, passwordHash, id)
	if err != nil {
		return fmt.Errorf("db: actualizando password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ClaimByOAuth se usa cuando un login por OAuth prueba, ahora mismo, que
// quien está del otro lado es el dueño real de un email que ya existía en
// la base pero sin verificar. Ese caso pasa cuando alguien registró esa
// dirección por email/password sin ser su dueño (a propósito o por error
// de tipeo) y todavía no la verificó: si el login por OAuth simplemente
// devolviera esa cuenta tal cual, el dueño original del password seguiría
// pudiendo entrar a la misma cuenta que el dueño real del email acaba de
// "reclamar" — un hueco real de pre-account-takeover. Por eso, además de
// marcar el email como verificado, se borra el password_hash viejo (deja
// de servir) y se actualiza el auth_provider.
func (r *UserRepo) ClaimByOAuth(ctx context.Context, id int64, provider models.AuthProvider) error {
	const q = `
		UPDATE users
		SET email_verified = true, password_hash = NULL, auth_provider = $1
		WHERE id = $2
	`
	tag, err := r.pool.Exec(ctx, q, provider, id)
	if err != nil {
		return fmt.Errorf("db: reclamando cuenta: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepo) scanOne(row pgx.Row) (models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.AuthProvider, &u.EmailVerified, &u.Role, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrNotFound
		}
		return models.User{}, fmt.Errorf("db: leyendo usuario: %w", err)
	}
	return u, nil
}
