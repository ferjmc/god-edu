package email

import "fmt"

// VerificationEmail arma el asunto y el HTML del correo de verificación
// de cuenta. link debe apuntar a la página del frontend que lee el token
// de la URL y llama a POST /auth/verify-email (esa página todavía no
// existe, se construye en la Fase 2 del frontend).
func VerificationEmail(link string) (subject, html string) {
	subject = "Confirmá tu cuenta"
	html = fmt.Sprintf(`
		<p>¡Hola!</p>
		<p>Confirmá tu cuenta haciendo clic en el siguiente enlace:</p>
		<p><a href="%s">Confirmar mi cuenta</a></p>
		<p>Si no creaste esta cuenta, podés ignorar este correo.</p>
		<p>Este enlace vence en 24 horas.</p>
	`, link)
	return subject, html
}

// PasswordResetEmail arma el asunto y el HTML del correo de reset de
// password. link debe apuntar a la página del frontend que lee el token
// de la URL y llama a POST /auth/reset-password.
func PasswordResetEmail(link string) (subject, html string) {
	subject = "Restablecé tu contraseña"
	html = fmt.Sprintf(`
		<p>¡Hola!</p>
		<p>Pediste restablecer tu contraseña. Hacé clic en el siguiente enlace para elegir una nueva:</p>
		<p><a href="%s">Restablecer contraseña</a></p>
		<p>Si no pediste esto, podés ignorar este correo: tu contraseña actual sigue siendo válida.</p>
		<p>Este enlace vence en 1 hora.</p>
	`, link)
	return subject, html
}
