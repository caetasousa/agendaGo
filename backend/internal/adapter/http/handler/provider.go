package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"agendago/internal/adapter/http/dto"
	"agendago/internal/domain/availability"
	"agendago/internal/domain/provider"
	ucauth "agendago/internal/usecase/auth"
	ucprovider "agendago/internal/usecase/provider"

	"github.com/go-playground/validator/v10"
)

// ProviderHandler concentra os handlers de prestador. identidadeDoContexto
// extrai a identidade posta no contexto pelo middleware de autenticação —
// recebida como função para evitar um import cycle entre os pacotes handler
// e middleware (mesmo padrão do AuthHandler).
type ProviderHandler struct {
	cadastrar             *ucprovider.CadastrarUseCase
	atualizarPreferencias *ucprovider.AtualizarPreferenciasUseCase
	identidadeDoContexto  func(r *http.Request) (ucauth.Identidade, bool)
}

func NovoProviderHandler(
	cadastrar *ucprovider.CadastrarUseCase,
	atualizarPreferencias *ucprovider.AtualizarPreferenciasUseCase,
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool),
) *ProviderHandler {
	return &ProviderHandler{
		cadastrar:             cadastrar,
		atualizarPreferencias: atualizarPreferencias,
		identidadeDoContexto:  identidadeDoContexto,
	}
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

// AtualizarPreferencias godoc
//
//	@Summary		Atualizar preferências do prestador
//	@Description	Atualiza as preferências de agenda do prestador autenticado
//	@Tags			providers
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.AtualizarPreferenciasRequest	true	"Preferências"
//	@Success		200		{object}	dto.AtualizarPreferenciasResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/providers/me/preferencias [put]
func (h *ProviderHandler) AtualizarPreferencias(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	var req dto.AtualizarPreferenciasRequest
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

	horarios := make([]ucprovider.BlocoInput, 0, len(req.HorariosPadrao))
	for _, b := range req.HorariosPadrao {
		horarios = append(horarios, ucprovider.BlocoInput{InicioMinutos: b.InicioMinutos, FimMinutos: b.FimMinutos})
	}

	output, err := h.atualizarPreferencias.Executar(ucprovider.AtualizarPreferenciasInput{
		ProviderID:         id.UserID,
		AceitaAgendamentos: req.AceitaAgendamentos,
		DescansoMinutos:    req.DescansoMinutos,
		HorariosPadrao:     horarios,
	})
	if err != nil {
		switch {
		case errors.Is(err, provider.ErrDescansoInvalido),
			errors.Is(err, availability.ErrFimAntesDoInicio),
			errors.Is(err, availability.ErrForaDoDia),
			errors.Is(err, availability.ErrGranularidadeInvalida),
			errors.Is(err, availability.ErrBlocosSobrepostos):
			responderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ucprovider.ErrProviderNaoEncontrado):
			responderErro(w, http.StatusNotFound, err.Error())
		default:
			responderErro(w, http.StatusInternalServerError, "erro interno")
		}
		return
	}

	responderJSON(w, http.StatusOK, dto.AtualizarPreferenciasResponse{
		AceitaAgendamentos: output.AceitaAgendamentos,
		DescansoMinutos:    output.DescansoMinutos,
		HorariosPadrao:     blocosParaDTO(output.HorariosPadrao),
	})
}

func mensagemValidacao(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s é obrigatório", fe.Field())
	case "email":
		return fmt.Sprintf("%s deve ser um e-mail válido", fe.Field())
	case "min":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("%s deve ter no mínimo %s caracteres", fe.Field(), fe.Param())
		}
		return fmt.Sprintf("%s deve ser no mínimo %s", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s deve ter no máximo %s caracteres", fe.Field(), fe.Param())
	case "oneof":
		return fmt.Sprintf("%s deve ser um dos valores: %s", fe.Field(), fe.Param())
	case "datetime":
		return fmt.Sprintf("%s deve estar no formato AAAA-MM-DD", fe.Field())
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
