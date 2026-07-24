package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	ucauth "agendago/internal/usecase/auth"
)

// nomeCookieOAuthState é o cookie curto que guarda o state e o nonce do fluxo
// OAuth em andamento, para comparar com o que volta na querystring do
// callback (proteção contra CSRF) e validar o id_token (proteção contra
// replay). O tipo de conta (client/provider) NÃO vai num cookie — fica
// gravado no registro server-side do state (ver oauthstate.State.Publico),
// para o callback nunca precisar confiar num dado sem vínculo criptográfico
// com o state consumido. nomeCookieOAuthVoltar carrega o destino pós-login
// (?voltar=), o mesmo papel do parâmetro nas outras rotas de login — como o
// Google não devolve query arbitrária no callback, ele precisa sobreviver no
// cookie.
const (
	nomeCookieOAuthState  = "agendago_oauth_state"
	nomeCookieOAuthNonce  = "agendago_oauth_nonce"
	nomeCookieOAuthVoltar = "agendago_oauth_voltar"
	ttlCookieOAuth        = 10 * time.Minute
)

// OAuthHandler concentra os handlers de login social (Google). destinoPadrao
// é o caminho no frontend para onde o usuário volta após o login; em erro,
// volta para /login com uma querystring de erro.
type OAuthHandler struct {
	loginSocial  *ucauth.LoginSocialUseCase
	cookieSeguro bool
	frontendURL  string
}

// NovoOAuthHandler cria uma instância de OAuthHandler com o usecase de login
// social injetado.
func NovoOAuthHandler(loginSocial *ucauth.LoginSocialUseCase, cookieSeguro bool, frontendURL string) *OAuthHandler {
	return &OAuthHandler{loginSocial: loginSocial, cookieSeguro: cookieSeguro, frontendURL: frontendURL}
}

// GoogleStartClient godoc
//
//	@Summary		Iniciar login social do cliente com Google
//	@Description	Redireciona para a tela de consentimento do Google
//	@Tags			auth
//	@Success		302
//	@Router			/auth/client/google/start [get]
func (h *OAuthHandler) GoogleStartClient(w http.ResponseWriter, r *http.Request) {
	h.iniciar(w, r, ucauth.PublicoClient)
}

// GoogleStartProvider godoc
//
//	@Summary		Iniciar login social do prestador com Google
//	@Description	Redireciona para a tela de consentimento do Google
//	@Tags			auth
//	@Success		302
//	@Router			/auth/provider/google/start [get]
func (h *OAuthHandler) GoogleStartProvider(w http.ResponseWriter, r *http.Request) {
	h.iniciar(w, r, ucauth.PublicoProvider)
}

func (h *OAuthHandler) iniciar(w http.ResponseWriter, r *http.Request, publico ucauth.PublicoLoginSocial) {
	urlAutorizacao, stateTexto, nonce, err := h.loginSocial.Iniciar(publico)
	if err != nil {
		responderErroInterno(w, r, err)
		return
	}

	http.SetCookie(w, novoCookieOAuth(nomeCookieOAuthState, stateTexto, h.cookieSeguro))
	http.SetCookie(w, novoCookieOAuth(nomeCookieOAuthNonce, nonce, h.cookieSeguro))
	if voltar := caminhoInternoOuVazio(r.URL.Query().Get("voltar")); voltar != "" {
		http.SetCookie(w, novoCookieOAuth(nomeCookieOAuthVoltar, voltar, h.cookieSeguro))
	}
	http.Redirect(w, r, urlAutorizacao, http.StatusFound)
}

// caminhoInternoOuVazio devolve voltar só se for um caminho interno (começa
// com "/" e não com "//") — mesma regra do frontend, para nunca virar um open
// redirect a partir de uma querystring controlada pelo usuário.
func caminhoInternoOuVazio(voltar string) string {
	if voltar != "" && strings.HasPrefix(voltar, "/") && !strings.HasPrefix(voltar, "//") {
		return voltar
	}
	return ""
}

// GoogleCallback godoc
//
//	@Summary		Callback do login social com Google
//	@Description	Recebe o retorno do Google, cria/vincula a conta e inicia a sessão
//	@Tags			auth
//	@Success		302
//	@Router			/auth/google/callback [get]
func (h *OAuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie := valorCookieOAuth(r, nomeCookieOAuthState)
	nonce := valorCookieOAuth(r, nomeCookieOAuthNonce)
	voltar := caminhoInternoOuVazio(valorCookieOAuth(r, nomeCookieOAuthVoltar))
	limparCookiesOAuth(w, h.cookieSeguro)

	code := r.URL.Query().Get("code")
	stateRecebido := r.URL.Query().Get("state")

	out, err := h.loginSocial.Concluir(r.Context(), code, stateRecebido, stateCookie, nonce)
	if err != nil {
		switch {
		// ErrCredenciaisInvalidas aqui não é falha real: acontece quando a
		// mesma identidade Google já está vinculada a uma conta do OUTRO
		// tipo (ex.: logou como prestador, depois tenta "Sou cliente" com a
		// mesma conta Google) — o vínculo aponta para um ID que não existe
		// na tabela do tipo pedido. Mesma mensagem de ErrEmailJaCadastradoOutroTipo.
		case errors.Is(err, ucauth.ErrEmailJaCadastradoOutroTipo), errors.Is(err, ucauth.ErrCredenciaisInvalidas):
			h.redirecionarErro(w, r, "social_outro_tipo")
		case errors.Is(err, ucauth.ErrStateInvalido), errors.Is(err, ucauth.ErrEmailNaoVerificado), errors.Is(err, ucauth.ErrUsuarioInativo):
			h.redirecionarErro(w, r, "social")
		default:
			responderErroInterno(w, r, err)
		}
		return
	}

	http.SetCookie(w, novoCookieSessao(out.Token, out.ExpiraEm, h.cookieSeguro))
	destino := "/painel"
	if voltar != "" {
		destino = voltar
	}
	http.Redirect(w, r, h.frontendURL+destino, http.StatusFound)
}

// redirecionarErro leva o usuário de volta ao login com um código de erro na
// querystring, para o frontend mostrar uma mensagem apropriada (ver
// routes/login/+page.svelte). codigo nunca contém dado dinâmico — só os
// literais definidos aqui, então não há risco de refletir entrada do usuário.
func (h *OAuthHandler) redirecionarErro(w http.ResponseWriter, r *http.Request, codigo string) {
	http.Redirect(w, r, h.frontendURL+"/login?erro="+codigo, http.StatusFound)
}

func novoCookieOAuth(nome, valor string, seguro bool) *http.Cookie {
	return &http.Cookie{
		Name:     nome,
		Value:    valor,
		Path:     "/",
		MaxAge:   int(ttlCookieOAuth.Seconds()),
		HttpOnly: true,
		Secure:   seguro,
		SameSite: http.SameSiteLaxMode,
	}
}

func valorCookieOAuth(r *http.Request, nome string) string {
	c, err := r.Cookie(nome)
	if err != nil {
		return ""
	}
	v, err := url.QueryUnescape(c.Value)
	if err != nil {
		return ""
	}
	return v
}

func limparCookiesOAuth(w http.ResponseWriter, seguro bool) {
	for _, nome := range []string{nomeCookieOAuthState, nomeCookieOAuthNonce, nomeCookieOAuthVoltar} {
		http.SetCookie(w, &http.Cookie{
			Name:     nome,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   seguro,
			SameSite: http.SameSiteLaxMode,
		})
	}
}
