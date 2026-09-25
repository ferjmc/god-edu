// Package domain contiene la entidad AuthToken, el tipo TokenPurpose y el
// puerto (Repository) que infrastructure/postgres implementa. No conoce
// SQL ni HTTP.
package domain

import (
	"context"
	"time"
)

// TokenPurpose distingue para qué sirve un auth_token: no reusamos un
// token de verificación de email como si fuera de reset de password.
type TokenPurpose string

const (
	TokenPurposeEmailVerification TokenPurpose = "email_verification"
	TokenPurposePasswordReset     TokenPurpose = "password_reset"
)

// AuthToken es un token de un solo uso (link de verificación de email o de
// reset de password). Solo se guarda el hash del valor real — el valor
// crudo lo genera api/auth.GenerateToken y viaja únicamente en el link del
// email, nunca se persiste.
type AuthToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	Purpose   TokenPurpose
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// Repository es el puerto que expone infrastructure/postgres.
type Repository interface {
	// Create invalida los tokens sin usar del mismo propósito para ese
	// usuario y crea uno nuevo, en una sola transacción: así nunca queda más
	// de un link "vivo" al mismo tiempo.
	Create(ctx context.Context, userID int64, purpose TokenPurpose, tokenHash string, ttl time.Duration) (AuthToken, error)
	// GetValidByHash busca un token no usado y no expirado, del propósito
	// dado. Devuelve ErrNotFound (db.ErrNotFound) sin distinguir si es
	// porque no existe, ya se usó o expiró.
	GetValidByHash(ctx context.Context, tokenHash string, purpose TokenPurpose) (AuthToken, error)
	// MarkUsed marca un token como consumido para que no se pueda reusar.
	MarkUsed(ctx context.Context, id int64) error
}
