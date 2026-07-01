package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"agendago/internal/adapter/http/dto"
	ucprovider "agendago/internal/usecase/provider"

	"github.com/go-playground/validator/v10"
)

type ProviderHandler struct {
	cadastrar *ucprovider.CadastrarUseCase
}

func NovoProviderHandler(cadastrar *ucprovider.CadastrarUseCase) *ProviderHandler {
	return &ProviderHandler{cadastrar: cadastrar}
}

// Cadastrar godoc
//
//	@Summary		Cadastrar prestador
//	@Description	Cria um novo prestador de serviço
//	@Tags			providers
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.CadastrarProviderRequest	true	"Dados do prestador"
//	@Success		201		{object}	dto.CadastrarProviderResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Router			/providers [post]
func (h *ProviderHandler) Cadastrar(w http.ResponseWriter, r *http.Request) {
	var req dto.CadastrarProviderRequest
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

	output, err := h.cadastrar.Executar(ucprovider.CadastrarInput{
		Nome:  req.Nome,
		Email: req.Email,
		Senha: req.Senha,
	})
	if err != nil {
		switch {
		case errors.Is(err, ucprovider.ErrEmailJaCadastrado):
			responderErro(w, http.StatusConflict, err.Error())
		default:
			responderErro(w, http.StatusInternalServerError, "erro interno")
		}
		return
	}

	responderJSON(w, http.StatusCreated, dto.CadastrarProviderResponse{
		ID:    output.ID,
		Nome:  output.Nome,
		Email: output.Email,
	})
}

func mensagemValidacao(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s é obrigatório", fe.Field())
	case "email":
		return fmt.Sprintf("%s deve ser um e-mail válido", fe.Field())
	case "min":
		return fmt.Sprintf("%s deve ter no mínimo %s caracteres", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s deve ter no máximo %s caracteres", fe.Field(), fe.Param())
	default:
		return fmt.Sprintf("%s é inválido", fe.Field())
	}
}

func responderJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func responderErro(w http.ResponseWriter, status int, msg string) {
	responderJSON(w, status, map[string]string{"erro": msg})
}
