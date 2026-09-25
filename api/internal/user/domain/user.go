// Package domain contiene la entidad User, los tipos AuthProvider/UserRole
// y el puerto (Repository) que infrastructure/postgres implementa. No
// conoce SQL ni HTTP.
package domain

import (
	"context"
	"errors"
	"time"
)

// ErrLastAdmin indica que la operación pedida dejaría a la plataforma sin
// ningún usuario con rol ADMIN.
var ErrLastAdmin = errors.New("user: no se puede quitar el último admin")

// ErrSelfRoleChange indica que un usuario intentó cambiar su propio rol.
// Existe como sentinel de dominio (no solo un mensaje en el handler) para
// que Service.UpdateRole sea el único lugar que decide esta regla — así
// cualquier caller futuro que llame al service (no solo el handler HTTP de
// hoy) queda protegido igual, sin tener que acordarse de repetir el chequeo.
var ErrSelfRoleChange = errors.New("user: no se puede cambiar el propio rol")

// AuthProvider identifica cómo se autenticó un usuario. Un usuario que se
// registró por email tiene PasswordHash; uno que vino por OAuth, no.
type AuthProvider string

const (
	AuthProviderEmail    AuthProvider = "email"
	AuthProviderGoogle   AuthProvider = "google"
	AuthProviderFacebook AuthProvider = "facebook"
)

// User es el registro de cuenta de un usuario de la plataforma.
type User struct {
	ID            int64
	Email         string
	PasswordHash  *string // nil si el usuario vino por OAuth
	Name          string
	AuthProvider  AuthProvider
	EmailVerified bool
	Role          UserRole
	CreatedAt     time.Time
}

// UserRole clasifica el nivel de acceso de un usuario. Un usuario tiene
// exactamente un rol (columna users.role); un curso puede restringir su
// detalle a un subconjunto arbitrario de roles (tabla
// course_visible_roles) — ver internal/course, Course.VisibleTo.
type UserRole string

const (
	RoleAdmin           UserRole = "ADMIN"
	RoleCommunityMember UserRole = "COMMUNITY_MEMBER"
	RolePublicMember    UserRole = "PUBLIC_MEMBER"
	RolePaidMember      UserRole = "PAID_MEMBER"
)

// Valid indica si role es uno de los cuatro roles conocidos. La usa
// AdminHandler.UpdateUserRole para rechazar un rol inventado antes de
// tocar la base (el CHECK constraint de la columna igual lo frenaría, pero
// devolver 400 con mensaje claro es mejor que un 500 de Postgres).
func (r UserRole) Valid() bool {
	switch r {
	case RoleAdmin, RoleCommunityMember, RolePublicMember, RolePaidMember:
		return true
	}
	return false
}

// Repository es el puerto que expone infrastructure/postgres.
type Repository interface {
	// ListAll devuelve todos los usuarios, más nuevos primero.
	ListAll(ctx context.Context) ([]User, error)
	// Create inserta un usuario nuevo y devuelve el registro con
	// id/created_at completados por la base.
	Create(ctx context.Context, u User) (User, error)
	// GetByEmail busca un usuario por email. Devuelve ErrNotFound si no
	// existe.
	GetByEmail(ctx context.Context, email string) (User, error)
	// GetByID busca un usuario por id. Devuelve ErrNotFound si no existe.
	GetByID(ctx context.Context, id int64) (User, error)
	// MarkEmailVerified pone email_verified = true para el usuario dado.
	MarkEmailVerified(ctx context.Context, id int64) error
	// UpdatePasswordHash reemplaza el password_hash de un usuario (flujo de
	// reset de password).
	UpdatePasswordHash(ctx context.Context, id int64, passwordHash string) error
	// ClaimByOAuth se usa cuando un login por OAuth prueba, ahora mismo, que
	// quien está del otro lado es el dueño real de un email que ya existía
	// en la base pero sin verificar (ver comentario completo en
	// infrastructure/postgres.Repository.ClaimByOAuth).
	ClaimByOAuth(ctx context.Context, id int64, provider AuthProvider) error
	// UpdateRole cambia el rol de un usuario. Operación cruda, sin chequeos
	// de negocio — esos viven en application.Service.UpdateRole, que es
	// quien decide si esta llamada procede.
	UpdateRole(ctx context.Context, id int64, role UserRole) error
	// CountAdmins devuelve cuántos usuarios tienen hoy el rol ADMIN. La usa
	// application.Service.UpdateRole para no dejar la plataforma sin
	// ningún admin.
	CountAdmins(ctx context.Context) (int, error)
}
