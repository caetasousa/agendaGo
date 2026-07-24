package config

import "os"

// GoogleClientID e GoogleClientSecret são as credenciais OAuth da aplicação
// no Google Cloud Console (env GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET).
func GoogleClientID() string     { return os.Getenv("GOOGLE_CLIENT_ID") }
func GoogleClientSecret() string { return os.Getenv("GOOGLE_CLIENT_SECRET") }

// GoogleRedirectURL é a URL de callback registrada no Google Cloud Console
// (env GOOGLE_REDIRECT_URL), para onde o Google devolve o usuário após o
// consentimento.
func GoogleRedirectURL() string { return os.Getenv("GOOGLE_REDIRECT_URL") }

// OAuthGoogleAtivo informa se o login social com Google está configurado.
// Quando falso, as rotas e o botão correspondente somem — evita expor um
// fluxo que não vai funcionar por falta de credenciais.
func OAuthGoogleAtivo() bool {
	return GoogleClientID() != "" && GoogleClientSecret() != "" && GoogleRedirectURL() != ""
}
