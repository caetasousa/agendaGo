package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"agendago/internal/adapter/http/dto"
	ucclient "agendago/internal/usecase/client"

	"github.com/go-playground/validator/v10"
)

type ClientHandler struct {
	cadastrar *ucclient.CadastrarUseCase
}

func NovoClientHandler(cadastrar *ucclient.CadastrarUseCase) *ClientHandler {
	return &ClientHandler{cadastrar: cadastrar}
}

// Cadastrar godoc
//
//	@Summary		Cadastrar cliente
//	@Description	Cria um novo cliente com conta
//	@Tags			clients
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.CadastrarClientRequest	true	"Dados do cliente"
//	@Success		201		{object}	dto.CadastrarClientResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Router			/clients [post]
func (h *ClientHandler) Cadastrar(w http.ResponseWriter, r *http.Request) {
	var req dto.CadastrarClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responderErro(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}

	if err := req.Validar(); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			responderErro(w, http.StatusBadRequest, mensagemValidacao(ve[0]))
			return
		}
		responderErro(w, http.StatusBadRequest, err.Error())
		return
	}

	output, err := h.cadastrar.Executar(ucclient.CadastrarInput{
		Nome:  req.Nome,
		Email: req.Email,
		Senha: req.Senha,
	})
	if err != nil {
		switch {
		case errors.Is(err, ucclient.ErrEmailJaCadastrado):
			responderErro(w, http.StatusConflict, err.Error())
		default:
			responderErro(w, http.StatusInternalServerError, "erro interno")
		}
		return
	}

	responderJSON(w, http.StatusCreated, dto.CadastrarClientResponse{
		ID:    output.ID,
		Nome:  output.Nome,
		Email: output.Email,
	})
}
