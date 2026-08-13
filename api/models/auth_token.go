package models

import "time"

// TokenPurpose distingue para qué sirve un auth_token: no reusamos un
// token de verificación de email como si fuera de reset de password.
type TokenPurpose string

const (
	TokenPurposeEmailVerification TokenPurpose = "email_verification"
	TokenPurposePasswordReset     TokenPurpose = "password_reset"
)

// AuthToken es un token de un solo uso (link de verificación de email o de
// reset de password). Solo se guarda el hash del valor real.
type AuthToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	Purpose   TokenPurpose
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
