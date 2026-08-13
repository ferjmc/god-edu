// Package email envía correos transaccionales (verificación de cuenta,
// reset de password) vía Resend.
package email

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

// Sender envía emails a través de la API de Resend.
type Sender struct {
	client *resend.Client
	from   string
}

// NewSender crea un Sender. from debe ser un remitente verificado en
// Resend, ej. "God Edu <no-reply@tudominio.org>".
func NewSender(apiKey, from string) *Sender {
	return &Sender{client: resend.NewClient(apiKey), from: from}
}

// Send manda un email HTML a un único destinatario.
func (s *Sender) Send(ctx context.Context, to, subject, html string) error {
	_, err := s.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: subject,
		Html:    html,
	})
	if err != nil {
		return fmt.Errorf("email: enviando a %s: %w", to, err)
	}
	return nil
}
