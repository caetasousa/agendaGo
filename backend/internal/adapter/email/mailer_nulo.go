package email

import "log/slog"

// MailerNulo é usado quando não há servidor SMTP configurado (config.EmailAtivo
// == false): em vez de falhar o boot, só registra no log o que seria enviado.
type MailerNulo struct{}

func (MailerNulo) Enviar(msg Mensagem) error {
	slog.Info("email: SMTP não configurado, mensagem descartada",
		slog.String("assunto", msg.Assunto), slog.String("para", msg.Para))
	return nil
}
