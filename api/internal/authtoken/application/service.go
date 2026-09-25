// Package application orquesta los casos de uso de tokens de un solo uso
// sobre el puerto domain.Repository. No conoce SQL ni HTTP.
package application

import (
	"context"
	"time"

	"github.com/ferjmc/god-edu/api/internal/authtoken/domain"
)

// Service es el punto de entrada del dominio authtoken para el resto de la
// API (hoy, los handlers en api/handlers). Wrapper delgado sobre el
// puerto — la invalidación de tokens previos ya es SQL transaccional
// dentro de Create, en infrastructure; no hay lógica de negocio que mover
// acá.
type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID int64, purpose domain.TokenPurpose, tokenHash string, ttl time.Duration) (domain.AuthToken, error) {
	return s.repo.Create(ctx, userID, purpose, tokenHash, ttl)
}

func (s *Service) GetValidByHash(ctx context.Context, tokenHash string, purpose domain.TokenPurpose) (domain.AuthToken, error) {
	return s.repo.GetValidByHash(ctx, tokenHash, purpose)
}

func (s *Service) MarkUsed(ctx context.Context, id int64) error {
	return s.repo.MarkUsed(ctx, id)
}
