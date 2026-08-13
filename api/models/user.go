package models

import "time"

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
