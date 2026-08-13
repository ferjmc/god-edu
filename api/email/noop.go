package email

import (
	"context"
	"log"
)

// NoopSender no manda nada: solo loguea. Se usa cuando RESEND_API_KEY no
// está configurada, para poder levantar y probar el resto de la API
// (registro, login) en desarrollo sin necesitar una cuenta de Resend.
type NoopSender struct{}

func (NoopSender) Send(ctx context.Context, to, subject, html string) error {
	log.Printf("email: RESEND_API_KEY no configurada, no se envió el correo a %s (asunto: %q)", to, subject)
	return nil
}
