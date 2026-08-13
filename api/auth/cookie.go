package auth

import "net/http"

// SessionCookieName es el nombre de la cookie que guarda el JWT de sesión.
const SessionCookieName = "god_edu_session"

// SetSessionCookie escribe el JWT en una cookie httpOnly, Secure,
// SameSite=Lax, como exige CLAUDE.md. Secure va siempre en true: los
// navegadores tratan a http://localhost como origen seguro, así que
// funciona también en desarrollo local (accediendo por "localhost", no
// por una IP de LAN).
func SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie borra la cookie de sesión (logout).
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
