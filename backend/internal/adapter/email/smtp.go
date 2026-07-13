package email

import (
	mail "github.com/wneessen/go-mail"
)

// MailerSMTP envia mensagens por um servidor SMTP real (Brevo em produção,
// Mailpit em desenvolvimento).
type MailerSMTP struct {
	host          string
	remetente     string
	remetenteNome string
	replyTo       string
	opcoes        []mail.Option
}

// NovaMailerSMTP cria um MailerSMTP. auth só é ativada quando usuário e
// senha não estão vazios — o Mailpit de desenvolvimento não exige
// autenticação. startTLS exige STARTTLS na conexão (TLSMandatory); quando
// falso, a conexão não usa TLS (caso do Mailpit). replyTo, se não vazio, vira
// o cabeçalho Reply-To das mensagens.
func NovaMailerSMTP(host string, porta int, usuario, senha string, startTLS bool, remetente, remetenteNome, replyTo string) (*MailerSMTP, error) {
	opcoes := []mail.Option{mail.WithPort(porta)}
	if startTLS {
		opcoes = append(opcoes, mail.WithTLSPolicy(mail.TLSMandatory))
	} else {
		opcoes = append(opcoes, mail.WithTLSPolicy(mail.NoTLS))
	}
	if usuario != "" && senha != "" {
		opcoes = append(opcoes,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(usuario),
			mail.WithPassword(senha),
		)
	}

	// valida a configuração cedo (host/porta/opções), embora a conexão de
	// fato só aconteça no envio de cada mensagem
	if _, err := mail.NewClient(host, opcoes...); err != nil {
		return nil, err
	}

	return &MailerSMTP{
		remetente:     remetente,
		remetenteNome: remetenteNome,
		replyTo:       replyTo,
		opcoes:        opcoes,
		host:          host,
	}, nil
}

// Enviar abre uma conexão SMTP e entrega a mensagem.
func (m *MailerSMTP) Enviar(msg Mensagem) error {
	cliente, err := mail.NewClient(m.host, m.opcoes...)
	if err != nil {
		return err
	}

	email := mail.NewMsg()
	if err := email.FromFormat(m.remetenteNome, m.remetente); err != nil {
		return err
	}
	if m.replyTo != "" {
		if err := email.ReplyTo(m.replyTo); err != nil {
			return err
		}
	}
	if err := email.To(msg.Para); err != nil {
		return err
	}
	email.Subject(msg.Assunto)
	email.SetBodyString(mail.TypeTextHTML, msg.HTML)

	return cliente.DialAndSend(email)
}
