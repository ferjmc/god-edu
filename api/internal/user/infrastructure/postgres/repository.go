// Package postgres implementa domain.Repository contra la tabla users.
// Única capa del dominio user que sabe que existe Postgres/pgx — mismo
// estilo (SQL crudo, sin ORM) que api/db.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ferjmc/god-edu/api/db"
	"github.com/ferjmc/god-edu/api/internal/user/domain"
)

// uniqueViolation es el código de error de Postgres para restricciones
// UNIQUE — mismo valor que db/users.go tenía, duplicado acá a propósito: es
// un código fijo del protocolo de Postgres, no algo que dependa del resto
// de la API.
const uniqueViolation = "23505"

// Repository implementa domain.Repository contra la tabla users. Los
// errores "no encontrado"/"ya existe" siguen siendo db.ErrNotFound y
// db.ErrConflict (no sentinels propios): son vocabulario compartido que ya
// usan handlers/respond.go (handleNotFound) y el resto de los handlers vía
// errors.Is — mantenerlos evita tocar esa capa solo por moverse de paquete.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListAll devuelve todos los usuarios, activos e inactivos, más nuevos
// primero. La usa el panel admin — necesita ver los inactivos para poder
// activarlos.
func (r *Repository) ListAll(ctx context.Context) ([]domain.User, error) {
	const q = `
		SELECT id, email, password_hash, name, auth_provider, email_verified, role, created_at
		FROM users
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("user: listando usuarios: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.AuthProvider, &u.EmailVerified, &u.Role, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("user: leyendo usuario: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("user: iterando usuarios: %w", err)
	}

	return users, nil
}

// Create inserta un usuario nuevo y devuelve el registro con id/created_at
// completados por la base.
func (r *Repository) Create(ctx context.Context, u domain.User) (domain.User, error) {
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
			return domain.User{}, db.ErrConflict
		}
		return domain.User{}, fmt.Errorf("user: creando usuario: %w", err)
	}

	return u, nil
}

// GetByEmail busca un usuario por email. Devuelve db.ErrNotFound si no
// existe.
func (r *Repository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	const q = `
		SELECT id, email, password_hash, name, auth_provider, email_verified, role, created_at
		FROM users
		WHERE email = $1
	`
	return r.scanOne(r.pool.QueryRow(ctx, q, email))
}

// GetByID busca un usuario por id. Devuelve db.ErrNotFound si no existe.
func (r *Repository) GetByID(ctx context.Context, id int64) (domain.User, error) {
	const q = `
		SELECT id, email, password_hash, name, auth_provider, email_verified, role, created_at
		FROM users
		WHERE id = $1
	`
	return r.scanOne(r.pool.QueryRow(ctx, q, id))
}

// MarkEmailVerified pone email_verified = true para el usuario dado.
func (r *Repository) MarkEmailVerified(ctx context.Context, id int64) error {
	const q = `UPDATE users SET email_verified = true WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("user: marcando email verificado: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return db.ErrNotFound
	}
	return nil
}

// UpdatePasswordHash reemplaza el password_hash de un usuario (flujo de
// reset de password).
func (r *Repository) UpdatePasswordHash(ctx context.Context, id int64, passwordHash string) error {
	const q = `UPDATE users SET password_hash = $1 WHERE id = $2`
	tag, err := r.pool.Exec(ctx, q, passwordHash, id)
	if err != nil {
		return fmt.Errorf("user: actualizando password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return db.ErrNotFound
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
func (r *Repository) ClaimByOAuth(ctx context.Context, id int64, provider domain.AuthProvider) error {
	const q = `
		UPDATE users
		SET email_verified = true, password_hash = NULL, auth_provider = $1
		WHERE id = $2
	`
	tag, err := r.pool.Exec(ctx, q, provider, id)
	if err != nil {
		return fmt.Errorf("user: reclamando cuenta: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return db.ErrNotFound
	}
	return nil
}

// UpdateRole cambia el rol crudo de un usuario, sin chequeos de negocio —
// esos viven en application.Service.UpdateRole. Tan simple como
// SetPublished en internal/course.
func (r *Repository) UpdateRole(ctx context.Context, id int64, role domain.UserRole) error {
	const q = `UPDATE users SET role = $1 WHERE id = $2`
	tag, err := r.pool.Exec(ctx, q, role, id)
	if err != nil {
		return fmt.Errorf("user: actualizando rol de usuario %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return db.ErrNotFound
	}
	return nil
}

// CountAdmins devuelve cuántos usuarios tienen hoy el rol ADMIN. La usa
// application.Service.UpdateRole para no dejar la plataforma sin ningún
// admin — antes era una query inline dentro del UpdateRole de db/users.go,
// extraída acá a su propio método porque la decisión de negocio que la usa
// ahora vive en application.
func (r *Repository) CountAdmins(ctx context.Context) (int, error) {
	const q = `SELECT COUNT(*) FROM users WHERE role = $1`
	var count int
	if err := r.pool.QueryRow(ctx, q, domain.RoleAdmin).Scan(&count); err != nil {
		return 0, fmt.Errorf("user: contando administradores: %w", err)
	}
	return count, nil
}

func (r *Repository) scanOne(row pgx.Row) (domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.AuthProvider, &u.EmailVerified, &u.Role, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, db.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("user: leyendo usuario: %w", err)
	}
	return u, nil
}
