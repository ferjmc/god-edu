package handlers

import (
	"net/http"

	"github.com/ferjmc/god-edu/api/auth"
	"github.com/ferjmc/god-edu/api/db"
	"github.com/ferjmc/god-edu/api/models"
)

// RequireAdmin es un middleware de chi que exige rol ADMIN. Se monta
// después de auth.RequireAuth (necesita el user_id que ese middleware deja
// en el contexto) — nunca antes, o UserIDFromContext no encuentra nada.
func RequireAdmin(users *db.UserRepo) func(http.Handler) http.Handler {
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

			if user.Role != models.RoleAdmin {
				writeError(w, http.StatusForbidden, "requiere rol de administrador")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
