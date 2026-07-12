// Package email implementa o envio de notificações por email: recuperação
// de senha e eventos de agendamento. O transporte real é SMTP (MailerSMTP),
// com um fake em memória para testes e um mailer nulo para quando não há
// servidor configurado.
package email

// Mensagem é um email pronto para envio, já com o HTML renderizado.
type Mensagem struct {
	Para     string
	NomePara string
	Assunto  string
	HTML     string
}

// enviador transporta uma Mensagem já pronta até o destinatário.
type enviador interface {
	Enviar(m Mensagem) error
}
