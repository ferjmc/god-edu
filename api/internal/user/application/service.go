// Package application orquesta los casos de uso de usuarios sobre el
// puerto domain.Repository. No conoce SQL ni HTTP.
package application

import (
	"context"

	"github.com/ferjmc/god-edu/api/internal/user/domain"
)

// Service es el punto de entrada del dominio user para el resto de la API
// (hoy, los handlers en api/handlers). Wrapper delgado para casi todo,
// excepto UpdateRole (ver abajo).
type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListAll(ctx context.Context) ([]domain.User, error) {
	return s.repo.ListAll(ctx)
}

func (s *Service) Create(ctx context.Context, u domain.User) (domain.User, error) {
	return s.repo.Create(ctx, u)
}

func (s *Service) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	return s.repo.GetByEmail(ctx, email)
}

func (s *Service) GetByID(ctx context.Context, id int64) (domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) MarkEmailVerified(ctx context.Context, id int64) error {
	return s.repo.MarkEmailVerified(ctx, id)
}

func (s *Service) UpdatePasswordHash(ctx context.Context, id int64, passwordHash string) error {
	return s.repo.UpdatePasswordHash(ctx, id, passwordHash)
}

func (s *Service) ClaimByOAuth(ctx context.Context, id int64, provider domain.AuthProvider) error {
	return s.repo.ClaimByOAuth(ctx, id, provider)
}

// UpdateRole cambia el rol de un usuario, con las dos protecciones que
// hasta esta migración vivían partidas entre el handler HTTP y el repo:
// (1) un caller no puede cambiar su propio rol por acá (evita
// autodegradarse sin querer), y (2) no se puede bajar de ADMIN al último
// administrador que queda, sin importar quién haga el cambio — esto último
// cubre el caso que (1) solo no alcanza a prevenir (un admin le baja el rol
// a otro admin, dejando cero). Consolidar ambas acá, en vez de dejar (1) en
// el handler, cierra un hueco real: antes, un caller que llamara al repo
// directo (otro handler, un script) se salteaba la protección (1) sin
// darse cuenta. Ambas protecciones se pueden saltar a mano contra la base
// si hace falta.
func (s *Service) UpdateRole(ctx context.Context, callerID, targetID int64, role domain.UserRole) error {
	if callerID == targetID {
		return domain.ErrSelfRoleChange
	}

	current, err := s.repo.GetByID(ctx, targetID)
	if err != nil {
		return err
	}

	if current.Role == domain.RoleAdmin && role != domain.RoleAdmin {
		// Nota deliberada: el chequeo (CountAdmins) y el UPDATE son dos
		// llamadas separadas al repo, no una transacción con lock — hay una
		// ventana teórica de carrera (dos requests concurrentes degradando
		// admins distintos podrían contar 2 cada uno y dejar el sistema sin
		// ningún admin). Se acepta ese riesgo a propósito: esta es una
		// plataforma de un solo desarrollador con manejo de admins manual y
		// de bajo volumen (ver CLAUDE.md, "simplicidad sobre elegancia") —
		// la alternativa correcta requiere transacción + row locks,
		// complejidad real para un caso que en la práctica no va a pasar. Si
		// algún día se necesitara cerrar la ventana del todo, ahí sí vale la
		// pena.
		count, err := s.repo.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return domain.ErrLastAdmin
		}
	}

	return s.repo.UpdateRole(ctx, targetID, role)
}
