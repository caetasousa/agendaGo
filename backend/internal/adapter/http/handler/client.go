package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"agendago/internal/adapter/http/dto"
	ucclient "agendago/internal/usecase/client"
)

// ClientHandler concentra o cadastro de cliente com verificação por email.
type ClientHandler struct {
	solicitarCadastro *ucclient.SolicitarCadastroUseCase
	confirmarCadastro *ucclient.ConfirmarCadastroUseCase
}

// NovoClientHandler cria uma instância de ClientHandler com os usecases injetados.
func NovoClientHandler(solicitarCadastro *ucclient.SolicitarCadastroUseCase, confirmarCadastro *ucclient.ConfirmarCadastroUseCase) *ClientHandler {
	return &ClientHandler{solicitarCadastro: solicitarCadastro, confirmarCadastro: confirmarCadastro}
}

// Cadastrar godoc
//
//	@Summary		Solicitar cadastro de cliente
//	@Description	Inicia o cadastro e envia um email de confirmação. Responde sempre 204 — exista ou não o email — para não revelar quais emails estão cadastrados.
//	@Tags			clients
//	@Accept			json
//	@Param			body	body	dto.CadastrarClientRequest	true	"Dados do cliente"
//	@Success		204
//	@Failure		400	{object}	map[string]string
//	@Router			/clients [post]
func (h *ClientHandler) Cadastrar(w http.ResponseWriter, r *http.Request) {
	var req dto.CadastrarClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responderErro(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	if err := req.Validar(); err != nil {
		responderErroValidacao(w, err)
		return
	}

	if err := h.solicitarCadastro.Executar(ucclient.SolicitarCadastroInput{
		Nome:     req.Nome,
		Email:    req.Email,
		Telefone: req.Telefone,
		Senha:    req.Senha,
	}); err != nil {
		responderErro(w, http.StatusInternalServerError, "erro interno")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ConfirmarCadastro godoc
//
//	@Summary		Confirmar cadastro de cliente
//	@Description	Conclui o cadastro a partir do token do email; cria a conta (ou converte um convidado, herdando os agendamentos)
//	@Tags			clients
//	@Accept			json
//	@Param			body	body	dto.ConfirmarCadastroRequest	true	"Token de confirmação"
//	@Success		204
//	@Failure		400	{object}	map[string]string
//	@Router			/clients/confirmar-cadastro [post]
func (h *ClientHandler) ConfirmarCadastro(w http.ResponseWriter, r *http.Request) {
	var req dto.ConfirmarCadastroRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responderErro(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	if err := req.Validar(); err != nil {
		responderErroValidacao(w, err)
		return
	}

	if _, err := h.confirmarCadastro.Executar(req.Token); err != nil {
		if errors.Is(err, ucclient.ErrCadastroInvalido) {
			responderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		responderErro(w, http.StatusInternalServerError, "erro interno")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
