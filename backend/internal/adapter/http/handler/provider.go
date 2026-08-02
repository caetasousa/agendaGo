package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"

	"agendago/internal/adapter/http/dto"
	"agendago/internal/domain/availability"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/usuario"
	"agendago/internal/pkg/logging"
	ucauth "agendago/internal/usecase/auth"
	ucprovider "agendago/internal/usecase/provider"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

// ProviderHandler concentra os handlers de prestador. identidadeDoContexto
// extrai a identidade posta no contexto pelo middleware de autenticação —
// recebida como função para evitar um import cycle entre os pacotes handler
// e middleware (mesmo padrão do AuthHandler).
type ProviderHandler struct {
	solicitarCadastro     *ucprovider.SolicitarCadastroUseCase
	confirmarCadastro     *ucprovider.ConfirmarCadastroUseCase
	atualizarPreferencias *ucprovider.AtualizarPreferenciasUseCase
	listar                *ucprovider.ListarUseCase
	buscarResumo          *ucprovider.BuscarResumoUseCase
	identidadeDoContexto  func(r *http.Request) (ucauth.Identidade, bool)
}

func NovoProviderHandler(
	solicitarCadastro *ucprovider.SolicitarCadastroUseCase,
	confirmarCadastro *ucprovider.ConfirmarCadastroUseCase,
	atualizarPreferencias *ucprovider.AtualizarPreferenciasUseCase,
	listar *ucprovider.ListarUseCase,
	buscarResumo *ucprovider.BuscarResumoUseCase,
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool),
) *ProviderHandler {
	return &ProviderHandler{
		solicitarCadastro:     solicitarCadastro,
		confirmarCadastro:     confirmarCadastro,
		atualizarPreferencias: atualizarPreferencias,
		listar:                listar,
		buscarResumo:          buscarResumo,
		identidadeDoContexto:  identidadeDoContexto,
	}
}

// Cadastrar godoc
//
//	@Summary		Solicitar cadastro de prestador
//	@Description	Inicia o cadastro e envia um email de confirmação. Responde sempre 204 — exista ou não o email — para não revelar quais emails estão cadastrados. A conta só nasce na confirmação.
//	@Tags			providers
//	@Accept			json
//	@Param			body	body	dto.CadastrarProviderRequest	true	"Dados do prestador"
//	@Success		204
//	@Failure		400	{object}	map[string]string
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

	if err := h.solicitarCadastro.Executar(ucprovider.SolicitarCadastroInput{
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

// ConfirmarCadastro godoc
//
//	@Summary		Confirmar cadastro de prestador
//	@Description	Conclui o cadastro a partir do token do email e cria a conta do prestador
//	@Tags			providers
//	@Accept			json
//	@Param			body	body	dto.ConfirmarCadastroRequest	true	"Token de confirmação"
//	@Success		204
//	@Failure		400	{object}	map[string]string
//	@Router			/providers/confirmar-cadastro [post]
func (h *ProviderHandler) ConfirmarCadastro(w http.ResponseWriter, r *http.Request) {
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
		if errors.Is(err, ucprovider.ErrCadastroInvalido) {
			responderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		responderErroInterno(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
		UsuarioID:                    id.UserID,
		ProviderID:                   id.ProviderID,
		Telefone:                     req.Telefone,
		AceitaAgendamentos:           req.AceitaAgendamentos,
		DescansoMinutos:              req.DescansoMinutos,
		DuracaoAtendimentoMinutos:    req.DuracaoAtendimentoMinutos,
		HorariosPadrao:               horarios,
		PermiteMarcacaoPeloPrestador: req.PermiteMarcacaoPeloPrestador,
		Slug:                         req.Slug,
	})
	if err != nil {
		switch {
		case errors.Is(err, provider.ErrDescansoInvalido),
			errors.Is(err, provider.ErrDuracaoInvalida),
			errors.Is(err, usuario.ErrTelefoneObrigatorio),
			errors.Is(err, availability.ErrFimAntesDoInicio),
			errors.Is(err, availability.ErrForaDoDia),
			errors.Is(err, availability.ErrGranularidadeInvalida),
			errors.Is(err, availability.ErrBlocosSobrepostos):
			responderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ucprovider.ErrProviderNaoEncontrado):
			responderErro(w, http.StatusNotFound, err.Error())
		default:
			responderErroInterno(w, r, err)
		}
		return
	}

	responderJSON(w, http.StatusOK, dto.AtualizarPreferenciasResponse{
		Slug:                         output.Slug,
		Telefone:                     output.Telefone,
		AceitaAgendamentos:           output.AceitaAgendamentos,
		DescansoMinutos:              output.DescansoMinutos,
		DuracaoAtendimentoMinutos:    output.DuracaoAtendimentoMinutos,
		HorariosPadrao:               blocosParaDTO(output.HorariosPadrao),
		PermiteMarcacaoPeloPrestador: output.PermiteMarcacaoPeloPrestador,
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

// responderErroInterno loga o erro real (com request_id e rota, para
// correlação e diagnóstico) e responde 500 genérico. O erro nunca vai para o
// corpo da resposta — só para o log —, então em produção dá para descobrir
// por que uma requisição falhou sem vazar detalhes internos ao cliente.
func responderErroInterno(w http.ResponseWriter, r *http.Request, err error) {
	logging.RequisicaoLogger(r).Error("erro interno no handler", slog.String("erro", err.Error()))
	responderErro(w, http.StatusInternalServerError, "erro interno")
}

// Listar godoc
//
//	@Summary		Listar prestadores
//	@Description	Lista uma página da vitrine; quem está com a agenda desativada aparece sem horários
//	@Tags			providers
//	@Produce		json
//	@Param			limite	query		int	false	"Itens por página (padrão 100, máximo 200)"
//	@Param			offset	query		int	false	"Deslocamento a partir do início da lista"
//	@Success		200		{object}	dto.ListarPrestadoresResponse
//	@Router			/providers [get]
func (h *ProviderHandler) Listar(w http.ResponseWriter, r *http.Request) {
	pag := paginaDaQuery(r)
	out, err := h.listar.Executar(pag)
	if err != nil {
		responderErroInterno(w, r, err)
		return
	}

	prestadores := make([]dto.PrestadorResumoDTO, 0, len(out.Prestadores))
	for _, p := range out.Prestadores {
		prestadores = append(prestadores, resumoParaDTO(p))
	}

	responderJSON(w, http.StatusOK, dto.ListarPrestadoresResponse{
		Prestadores:  prestadores,
		PaginacaoDTO: dto.NovaPaginacao(pag, out.Total),
	})
}

// BuscarResumo godoc
//
//	@Summary		Buscar prestador
//	@Description	Devolve a identificação pública de um prestador — usada pela página de agendamento acessada via link direto
//	@Tags			providers
//	@Produce		json
//	@Param			id	path		string	true	"ID do prestador"
//	@Success		200	{object}	dto.PrestadorResumoDTO
//	@Failure		404	{object}	map[string]string
//	@Router			/providers/{id} [get]
func (h *ProviderHandler) BuscarResumo(w http.ResponseWriter, r *http.Request) {
	resumo, err := h.buscarResumo.Executar(chi.URLParam(r, "id"))
	if err != nil {
		switch {
		case errors.Is(err, ucprovider.ErrProviderNaoEncontrado):
			responderErro(w, http.StatusNotFound, err.Error())
		default:
			responderErroInterno(w, r, err)
		}
		return
	}

	responderJSON(w, http.StatusOK, resumoParaDTO(*resumo))
}

func resumoParaDTO(p ucprovider.PrestadorResumo) dto.PrestadorResumoDTO {
	return dto.PrestadorResumoDTO{
		ID:                        p.ID,
		Slug:                      p.Slug,
		Nome:                      p.Nome,
		DuracaoAtendimentoMinutos: p.DuracaoAtendimentoMinutos,
		AceitaAgendamentos:        p.AceitaAgendamentos,
	}
}
