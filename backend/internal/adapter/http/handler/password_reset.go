package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"agendago/internal/adapter/http/dto"
	ucauth "agendago/internal/usecase/auth"

	"github.com/go-playground/validator/v10"
)

// PasswordResetHandler concentra os handlers de recuperação de senha.
type PasswordResetHandler struct {
	solicitar *ucauth.SolicitarRecuperacaoUseCase
	redefinir *ucauth.RedefinirSenhaUseCase
}

// NovoPasswordResetHandler cria uma instância de PasswordResetHandler com os usecases injetados.
func NovoPasswordResetHandler(solicitar *ucauth.SolicitarRecuperacaoUseCase, redefinir *ucauth.RedefinirSenhaUseCase) *PasswordResetHandler {
	return &PasswordResetHandler{solicitar: solicitar, redefinir: redefinir}
}

// Solicitar godoc
//
//	@Summary		Solicitar recuperação de senha
//	@Description	Envia por email um link de redefinição de senha, se o email pertencer a uma conta. A resposta é sempre 204, exista ou não a conta, para não revelar quais emails estão cadastrados.
//	@Tags			auth
//	@Accept			json
//	@Param			body	body	dto.RecuperarSenhaRequest	true	"Email da conta"
//	@Success		204
//	@Failure		400	{object}	map[string]string
//	@Router			/auth/recuperar-senha [post]
func (h *PasswordResetHandler) Solicitar(w http.ResponseWriter, r *http.Request) {
	var req dto.RecuperarSenhaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responderErro(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	if err := req.Validar(); err != nil {
		responderErroValidacao(w, err)
		return
	}

	if err := h.solicitar.Executar(req.Email); err != nil {
		responderErroInterno(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Redefinir godoc
//
//	@Summary		Redefinir senha
//	@Description	Troca a senha usando um token de recuperação válido e ainda não utilizado
//	@Tags			auth
//	@Accept			json
//	@Param			body	body	dto.RedefinirSenhaRequest	true	"Token e nova senha"
//	@Success		204
//	@Failure		400	{object}	map[string]string
//	@Router			/auth/redefinir-senha [post]
func (h *PasswordResetHandler) Redefinir(w http.ResponseWriter, r *http.Request) {
	var req dto.RedefinirSenhaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responderErro(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	if err := req.Validar(); err != nil {
		responderErroValidacao(w, err)
		return
	}

	if err := h.redefinir.Executar(req.Token, req.NovaSenha); err != nil {
		if errors.Is(err, ucauth.ErrTokenRecuperacaoInvalido) {
			responderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		responderErroInterno(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func responderErroValidacao(w http.ResponseWriter, err error) {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		responderErro(w, http.StatusBadRequest, mensagemValidacao(ve[0]))
		return
	}
	responderErro(w, http.StatusBadRequest, err.Error())
}
