package email

import "log"

// MailerNulo é usado quando não há servidor SMTP configurado (config.EmailAtivo
// == false): em vez de falhar o boot, só registra no log o que seria enviado.
type MailerNulo struct{}

func (MailerNulo) Enviar(msg Mensagem) error {
	log.Printf("email: SMTP não configurado, mensagem descartada (assunto=%q, para=%q)", msg.Assunto, msg.Para)
	return nil
}
