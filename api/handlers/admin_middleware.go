package handlers

import (
	"net/http"

	"github.com/ferjmc/god-edu/api/auth"
	userapp "github.com/ferjmc/god-edu/api/internal/user/application"
	userdomain "github.com/ferjmc/god-edu/api/internal/user/domain"
)

// RequireAdmin es un middleware de chi que exige rol ADMIN. Se monta
// después de auth.RequireAuth (necesita el user_id que ese middleware deja
// en el contexto) — nunca antes, o UserIDFromContext no encuentra nada.
func RequireAdmin(users *userapp.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := auth.UserIDFromContext(r.Context())
			if !ok {
				// No debería pasar: este middleware siempre va después de
				// auth.RequireAuth.
				writeError(w, http.StatusUnauthorized, "no autenticado")
				return
			}

			user, err := users.GetByID(r.Context(), userID)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "no autenticado")
				return
			}

			if user.Role != userdomain.RoleAdmin {
				writeError(w, http.StatusForbidden, "requiere rol de administrador")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
