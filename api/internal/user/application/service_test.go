package application

import (
	"context"
	"errors"
	"testing"

	"github.com/ferjmc/god-edu/api/db"
	"github.com/ferjmc/god-edu/api/internal/user/domain"
)

// fakeRepo implementa domain.Repository en memoria — solo lo que
// Service.UpdateRole necesita tiene lógica real (GetByID, UpdateRole,
// CountAdmins); el resto compila para satisfacer la interfaz pero no se
// ejercita en estos tests.
type fakeRepo struct {
	users map[int64]domain.User
}

func newFakeRepo(users ...domain.User) *fakeRepo {
	m := make(map[int64]domain.User, len(users))
	for _, u := range users {
		m[u.ID] = u
	}
	return &fakeRepo{users: m}
}

func (f *fakeRepo) ListAll(ctx context.Context) ([]domain.User, error) { return nil, nil }
func (f *fakeRepo) Create(ctx context.Context, u domain.User) (domain.User, error) {
	return domain.User{}, nil
}
func (f *fakeRepo) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	return domain.User{}, nil
}

func (f *fakeRepo) GetByID(ctx context.Context, id int64) (domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return domain.User{}, db.ErrNotFound
	}
	return u, nil
}

func (f *fakeRepo) MarkEmailVerified(ctx context.Context, id int64) error { return nil }
func (f *fakeRepo) UpdatePasswordHash(ctx context.Context, id int64, passwordHash string) error {
	return nil
}
func (f *fakeRepo) ClaimByOAuth(ctx context.Context, id int64, provider domain.AuthProvider) error {
	return nil
}

func (f *fakeRepo) UpdateRole(ctx context.Context, id int64, role domain.UserRole) error {
	u, ok := f.users[id]
	if !ok {
		return db.ErrNotFound
	}
	u.Role = role
	f.users[id] = u
	return nil
}

func (f *fakeRepo) CountAdmins(ctx context.Context) (int, error) {
	count := 0
	for _, u := range f.users {
		if u.Role == domain.RoleAdmin {
			count++
		}
	}
	return count, nil
}

func TestService_UpdateRole(t *testing.T) {
	const (
		soleAdmin   int64 = 1
		otherAdmin  int64 = 2
		normalUser  int64 = 3
		missingUser int64 = 99
	)

	newRepo := func() *fakeRepo {
		return newFakeRepo(
			domain.User{ID: soleAdmin, Role: domain.RoleAdmin},
			domain.User{ID: otherAdmin, Role: domain.RoleAdmin},
			domain.User{ID: normalUser, Role: domain.RolePublicMember},
		)
	}

	t.Run("auto-cambio de rol", func(t *testing.T) {
		repo := newRepo()
		svc := NewService(repo)

		err := svc.UpdateRole(context.Background(), soleAdmin, soleAdmin, domain.RolePublicMember)
		if !errors.Is(err, domain.ErrSelfRoleChange) {
			t.Fatalf("esperaba ErrSelfRoleChange, recibí %v", err)
		}
		if repo.users[soleAdmin].Role != domain.RoleAdmin {
			t.Fatal("el rol no debería haber cambiado")
		}
	})

	t.Run("usuario destino inexistente", func(t *testing.T) {
		repo := newRepo()
		svc := NewService(repo)

		err := svc.UpdateRole(context.Background(), soleAdmin, missingUser, domain.RolePublicMember)
		if !errors.Is(err, db.ErrNotFound) {
			t.Fatalf("esperaba db.ErrNotFound, recibí %v", err)
		}
	})

	t.Run("degradar al único admin", func(t *testing.T) {
		// Solo queda un admin (soleAdmin) en este repo — degradar al otro
		// admin (otherAdmin) sería dejar exactamente uno, no cero, así que
		// arrancamos con un repo de un solo admin para forzar el caso.
		repo := newFakeRepo(
			domain.User{ID: soleAdmin, Role: domain.RoleAdmin},
			domain.User{ID: normalUser, Role: domain.RolePublicMember},
		)
		svc := NewService(repo)

		err := svc.UpdateRole(context.Background(), normalUser, soleAdmin, domain.RolePublicMember)
		if !errors.Is(err, domain.ErrLastAdmin) {
			t.Fatalf("esperaba ErrLastAdmin, recibí %v", err)
		}
		if repo.users[soleAdmin].Role != domain.RoleAdmin {
			t.Fatal("el rol no debería haber cambiado")
		}
	})

	t.Run("degradar un admin habiendo otro", func(t *testing.T) {
		repo := newRepo() // soleAdmin + otherAdmin, ambos ADMIN
		svc := NewService(repo)

		err := svc.UpdateRole(context.Background(), soleAdmin, otherAdmin, domain.RolePublicMember)
		if err != nil {
			t.Fatalf("no esperaba error, recibí %v", err)
		}
		if repo.users[otherAdmin].Role != domain.RolePublicMember {
			t.Fatalf("esperaba que el rol cambiara, quedó %v", repo.users[otherAdmin].Role)
		}
	})

	t.Run("cambiar el rol de un usuario no-admin", func(t *testing.T) {
		repo := newRepo()
		svc := NewService(repo)

		err := svc.UpdateRole(context.Background(), soleAdmin, normalUser, domain.RolePaidMember)
		if err != nil {
			t.Fatalf("no esperaba error, recibí %v", err)
		}
		if repo.users[normalUser].Role != domain.RolePaidMember {
			t.Fatalf("esperaba RolePaidMember, quedó %v", repo.users[normalUser].Role)
		}
	})
}
