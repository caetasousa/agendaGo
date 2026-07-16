package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"agendago/internal/adapter/http/dto"
	ucclient "agendago/internal/usecase/client"

	"github.com/go-chi/chi/v5"
)

// ClientHandler concentra o cadastro de cliente com verificação por email.
type ClientHandler struct {
	solicitarCadastro    *ucclient.SolicitarCadastroUseCase
	confirmarCadastro    *ucclient.ConfirmarCadastroUseCase
	consultarPreCadastro *ucclient.ConsultarPreCadastroUseCase
	concluirPreCadastro  *ucclient.ConcluirPreCadastroUseCase
}

// NovoClientHandler cria uma instância de ClientHandler com os usecases injetados.
func NovoClientHandler(
	solicitarCadastro *ucclient.SolicitarCadastroUseCase,
	confirmarCadastro *ucclient.ConfirmarCadastroUseCase,
	consultarPreCadastro *ucclient.ConsultarPreCadastroUseCase,
	concluirPreCadastro *ucclient.ConcluirPreCadastroUseCase,
) *ClientHandler {
	return &ClientHandler{
		solicitarCadastro:    solicitarCadastro,
		confirmarCadastro:    confirmarCadastro,
		consultarPreCadastro: consultarPreCadastro,
		concluirPreCadastro:  concluirPreCadastro,
	}
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
		responderErroInterno(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ConsultarPreCadastro godoc
//
//	@Summary		Consultar dados de pré-cadastro
//	@Description	Rota pública: devolve nome/email/telefone do convidado a partir do token de pré-cadastro (uso único), para a tela de cadastro pré-preencher o formulário
//	@Tags			clients
//	@Produce		json
//	@Param			token	path		string	true	"Token de pré-cadastro"
//	@Success		200		{object}	dto.PreCadastroResponse
//	@Failure		400		{object}	map[string]string
//	@Router			/clients/pre-cadastro/{token} [get]
func (h *ClientHandler) ConsultarPreCadastro(w http.ResponseWriter, r *http.Request) {
	out, err := h.consultarPreCadastro.Executar(chi.URLParam(r, "token"))
	if err != nil {
		if errors.Is(err, ucclient.ErrPreCadastroInvalido) {
			responderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		responderErroInterno(w, r, err)
		return
	}

	responderJSON(w, http.StatusOK, dto.PreCadastroResponse{
		Nome:     out.Nome,
		Email:    out.Email,
		Telefone: out.Telefone,
	})
}

// ConcluirPreCadastro godoc
//
//	@Summary		Concluir cadastro a partir do pré-cadastro
//	@Description	Rota pública: cria a conta direto com a senha informada, sem uma segunda confirmação por email — quem tem o token de pré-cadastro já provou posse do email (recebido dentro do email de agendamento). Cria a conta nova ou converte um convidado existente, herdando os agendamentos
//	@Tags			clients
//	@Accept			json
//	@Produce		json
//	@Param			token	path		string							true	"Token de pré-cadastro"
//	@Param			body	body		dto.ConcluirPreCadastroRequest	true	"Senha da conta"
//	@Success		200		{object}	dto.ConcluirPreCadastroResponse
//	@Failure		400		{object}	map[string]string
//	@Router			/clients/pre-cadastro/{token} [post]
func (h *ClientHandler) ConcluirPreCadastro(w http.ResponseWriter, r *http.Request) {
	var req dto.ConcluirPreCadastroRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responderErro(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	if err := req.Validar(); err != nil {
		responderErroValidacao(w, err)
		return
	}

	out, err := h.concluirPreCadastro.Executar(ucclient.ConcluirPreCadastroInput{
		Token: chi.URLParam(r, "token"),
		Senha: req.Senha,
	})
	if err != nil {
		if errors.Is(err, ucclient.ErrPreCadastroInvalido) || errors.Is(err, ucclient.ErrCadastroInvalido) {
			responderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		responderErroInterno(w, r, err)
		return
	}

	responderJSON(w, http.StatusOK, dto.ConcluirPreCadastroResponse{Nome: out.Nome, Email: out.Email})
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
		responderErroInterno(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
