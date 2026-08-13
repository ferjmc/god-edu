package auth

import (
	"context"
	"encoding/json"
	"net/http"
)

type contextKey int

const userIDContextKey contextKey = iota

// RequireAuth es un middleware de chi que exige una cookie de sesión JWT
// válida. Si falta, es inválida o venció, corta con 401 antes de llegar al
// handler protegido. Si es válida, deja el user_id disponible en el
// contexto vía UserIDFromContext.
func RequireAuth(jwt *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "no autenticado")
				return
			}

			claims, err := jwt.Parse(cookie.Value)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "sesión inválida o vencida")
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext devuelve el user_id puesto por RequireAuth. El segundo
// valor es false si el middleware no corrió antes (no debería pasar en una
// ruta montada bajo RequireAuth).
func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDContextKey).(int64)
	return id, ok
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
