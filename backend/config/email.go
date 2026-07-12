package config

import "os"

// SMTPHost é o endereço do servidor SMTP (env SMTP_HOST). Vazio desliga o
// envio de email: o sistema sobe normalmente e usa um mailer que só loga,
// útil para boot e testes sem SMTP configurado.
func SMTPHost() string { return os.Getenv("SMTP_HOST") }

// SMTPPort é a porta do servidor SMTP (env SMTP_PORT, padrão 587).
func SMTPPort() int {
	return intDoAmbiente("SMTP_PORT", 587)
}

// SMTPUser e SMTPPassword são as credenciais de autenticação SMTP.
// Vazias desligam a autenticação (caso do Mailpit em desenvolvimento).
func SMTPUser() string     { return os.Getenv("SMTP_USER") }
func SMTPPassword() string { return os.Getenv("SMTP_PASSWORD") }

// SMTPStartTLS informa se a conexão exige STARTTLS (env SMTP_STARTTLS,
// padrão true). Desligado em desenvolvimento contra o Mailpit, que não
// fala TLS.
func SMTPStartTLS() bool {
	v := os.Getenv("SMTP_STARTTLS")
	if v == "" {
		return true
	}
	return v == "true" || v == "1"
}

// EmailRemetente e EmailRemetenteNome identificam o remetente dos emails
// enviados pelo sistema (precisa ser o email verificado no provedor SMTP).
func EmailRemetente() string     { return os.Getenv("EMAIL_REMETENTE") }
func EmailRemetenteNome() string { return os.Getenv("EMAIL_REMETENTE_NOME") }

// EmailAtivo informa se há um servidor SMTP configurado. Quando falso, o
// sistema usa um mailer nulo em vez de tentar enviar emails de verdade.
func EmailAtivo() bool { return SMTPHost() != "" }
