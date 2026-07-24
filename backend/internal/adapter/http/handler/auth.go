package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"agendago/internal/adapter/http/dto"
	"agendago/internal/pkg/logging"
	ucauth "agendago/internal/usecase/auth"

	"github.com/go-playground/validator/v10"
)

// AuthHandler concentra os handlers de login, logout e perfil autenticado.
// identidadeDoContexto extrai a identidade posta no contexto pelo middleware
// de autenticação — recebida como função para evitar um import cycle entre
// os pacotes handler e middleware.
type AuthHandler struct {
	loginProvider        *ucauth.LoginProviderUseCase
	loginClient          *ucauth.LoginClientUseCase
	loginAdmin           *ucauth.LoginAdminUseCase
	logout               *ucauth.LogoutUseCase
	perfil               *ucauth.PerfilUseCase
	cookieSeguro         bool
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool)
}

// NovoAuthHandler cria uma instância de AuthHandler com os usecases de autenticação injetados.
// identidadeDoContexto extrai a identidade posta no contexto pelo middleware de autenticação.
func NovoAuthHandler(
	loginProvider *ucauth.LoginProviderUseCase,
	loginClient *ucauth.LoginClientUseCase,
	loginAdmin *ucauth.LoginAdminUseCase,
	logout *ucauth.LogoutUseCase,
	perfil *ucauth.PerfilUseCase,
	cookieSeguro bool,
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool),
) *AuthHandler {
	return &AuthHandler{
		loginProvider:        loginProvider,
		loginClient:          loginClient,
		loginAdmin:           loginAdmin,
		logout:               logout,
		perfil:               perfil,
		cookieSeguro:         cookieSeguro,
		identidadeDoContexto: identidadeDoContexto,
	}
}

// LoginProvider godoc
//
//	@Summary		Login do prestador
//	@Description	Autentica um prestador e inicia uma sessão
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.LoginRequest	true	"Credenciais"
//	@Success		200		{object}	dto.LoginResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Router			/auth/provider/login [post]
func (h *AuthHandler) LoginProvider(w http.ResponseWriter, r *http.Request) {
	req, ok := decodificarLogin(w, r)
	if !ok {
		return
	}

	out, err := h.loginProvider.Executar(ucauth.LoginInput{Email: req.Email, Senha: req.Senha})
	if err != nil {
		responderErroLogin(w, r, "provider", req.Email, err)
		return
	}

	http.SetCookie(w, novoCookieSessao(out.Token, out.ExpiraEm, h.cookieSeguro))
	responderJSON(w, http.StatusOK, dto.LoginResponse{ID: out.UserID, Nome: out.Nome, Tipo: "provider"})
}

// LoginClient godoc
//
//	@Summary		Login do cliente
//	@Description	Autentica um cliente com conta e inicia uma sessão
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.LoginRequest	true	"Credenciais"
//	@Success		200		{object}	dto.LoginResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Router			/auth/client/login [post]
func (h *AuthHandler) LoginClient(w http.ResponseWriter, r *http.Request) {
	req, ok := decodificarLogin(w, r)
	if !ok {
		return
	}

	out, err := h.loginClient.Executar(ucauth.LoginInput{Email: req.Email, Senha: req.Senha})
	if err != nil {
		responderErroLogin(w, r, "client", req.Email, err)
		return
	}

	http.SetCookie(w, novoCookieSessao(out.Token, out.ExpiraEm, h.cookieSeguro))
	responderJSON(w, http.StatusOK, dto.LoginResponse{ID: out.UserID, Nome: out.Nome, Tipo: "client"})
}

// LoginAdmin godoc
//
//	@Summary		Login do administrador
//	@Description	Autentica um administrador e inicia uma sessão
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.LoginRequest	true	"Credenciais"
//	@Success		200		{object}	dto.LoginResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Router			/auth/admin/login [post]
func (h *AuthHandler) LoginAdmin(w http.ResponseWriter, r *http.Request) {
	req, ok := decodificarLogin(w, r)
	if !ok {
		return
	}

	out, err := h.loginAdmin.Executar(ucauth.LoginInput{Email: req.Email, Senha: req.Senha})
	if err != nil {
		responderErroLogin(w, r, "admin", req.Email, err)
		return
	}

	http.SetCookie(w, novoCookieSessao(out.Token, out.ExpiraEm, h.cookieSeguro))
	responderJSON(w, http.StatusOK, dto.LoginResponse{ID: out.UserID, Nome: out.Nome, Tipo: "admin"})
}

// Logout godoc
//
//	@Summary		Logout
//	@Description	Encerra a sessão atual
//	@Tags			auth
//	@Success		204
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(NomeCookieSessao); err == nil {
		h.logout.Executar(cookie.Value)
	}
	http.SetCookie(w, cookieSessaoExpirado(h.cookieSeguro))
	w.WriteHeader(http.StatusNoContent)
}

// Me godoc
//
//	@Summary		Usuário autenticado atual
//	@Description	Retorna os dados do usuário autenticado na sessão
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	dto.MeResponse
//	@Failure		401	{object}	map[string]string
//	@Router			/auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	perfil, err := h.perfil.Executar(id)
	if err != nil {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	responderJSON(w, http.StatusOK, dto.MeResponse{
		ID:                           perfil.ID,
		Nome:                         perfil.Nome,
		Email:                        perfil.Email,
		Telefone:                     perfil.Telefone,
		Tipo:                         perfil.Tipo,
		AceitaAgendamentos:           perfil.AceitaAgendamentos,
		DescansoMinutos:              perfil.DescansoMinutos,
		DuracaoAtendimentoMinutos:    perfil.DuracaoAtendimentoMinutos,
		HorariosPadrao:               blocosParaDTO(perfil.HorariosPadrao),
		PermiteMarcacaoPeloPrestador: perfil.PermiteMarcacaoPeloPrestador,
		TelefonePendente:             perfil.TelefonePendente,
	})
}

func decodificarLogin(w http.ResponseWriter, r *http.Request) (dto.LoginRequest, bool) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responderErro(w, http.StatusBadRequest, "corpo da requisição inválido")
		return req, false
	}

	if err := req.Validar(); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			responderErro(w, http.StatusBadRequest, mensagemValidacao(ve[0]))
			return req, false
		}
		responderErro(w, http.StatusBadRequest, err.Error())
		return req, false
	}

	return req, true
}

// responderErroLogin mapeia a falha de login para a resposta HTTP e registra
// o evento de segurança: credenciais inválidas e usuário banido saem em WARN
// (com tipo, email e IP) para permitir detectar brute-force e abuso; um erro
// inesperado cai no 500 logado. A senha nunca é logada.
func responderErroLogin(w http.ResponseWriter, r *http.Request, tipo, email string, err error) {
	switch {
	case errors.Is(err, ucauth.ErrCredenciaisInvalidas):
		logging.RequisicaoLogger(r).Warn("falha de login: credenciais inválidas",
			slog.String("tipo", tipo), slog.String("email", email), slog.String("ip", r.RemoteAddr))
		responderErro(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, ucauth.ErrUsuarioInativo):
		logging.RequisicaoLogger(r).Warn("login bloqueado: usuário banido",
			slog.String("tipo", tipo), slog.String("email", email), slog.String("ip", r.RemoteAddr))
		responderErro(w, http.StatusForbidden, err.Error())
	default:
		responderErroInterno(w, r, err)
	}
}
