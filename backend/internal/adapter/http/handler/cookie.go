package handler

import (
	"net/http"
	"time"
)

// NomeCookieSessao é o nome do cookie que carrega o token opaco de sessão.
const NomeCookieSessao = "agendago_session"

// novoCookieSessao monta o cookie de sessão: HttpOnly, SameSite=Lax, Path=/,
// Secure conforme o ambiente, e Max-Age derivado de expiraEm.
func novoCookieSessao(tokenPuro string, expiraEm time.Time, seguro bool) *http.Cookie {
	return &http.Cookie{
		Name:     NomeCookieSessao,
		Value:    tokenPuro,
		Path:     "/",
		Expires:  expiraEm,
		MaxAge:   int(time.Until(expiraEm).Seconds()),
		HttpOnly: true,
		Secure:   seguro,
		SameSite: http.SameSiteLaxMode,
	}
}

// cookieSessaoExpirado devolve um cookie com Max-Age negativo, usado para
// apagar o cookie de sessão do navegador no logout.
func cookieSessaoExpirado(seguro bool) *http.Cookie {
	return &http.Cookie{
		Name:     NomeCookieSessao,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   seguro,
		SameSite: http.SameSiteLaxMode,
	}
}
