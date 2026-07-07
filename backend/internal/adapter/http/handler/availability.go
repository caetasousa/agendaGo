package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"agendago/internal/adapter/http/dto"
	"agendago/internal/domain/availability"
	ucauth "agendago/internal/usecase/auth"
	ucavailability "agendago/internal/usecase/availability"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

// layoutData é o formato de data usado nos DTOs de disponibilidade (sem componente de hora).
const layoutData = "2006-01-02"

// AvailabilityHandler concentra os handlers de disponibilidade do prestador
// (grade semanal e exceções de data). identidadeDoContexto extrai a
// identidade posta no contexto pelo middleware de autenticação — recebida
// como função para evitar um import cycle entre os pacotes handler e middleware.
type AvailabilityHandler struct {
	definirGradeSemanal   *ucavailability.DefinirGradeSemanalUseCase
	consultarGradeSemanal *ucavailability.ConsultarGradeSemanalUseCase
	criarExcecao          *ucavailability.CriarExcecaoUseCase
	removerExcecao        *ucavailability.RemoverExcecaoUseCase
	listarExcecoes        *ucavailability.ListarExcecoesUseCase
	identidadeDoContexto  func(r *http.Request) (ucauth.Identidade, bool)
}

// NovoAvailabilityHandler cria uma instância de AvailabilityHandler com os usecases injetados.
func NovoAvailabilityHandler(
	definirGradeSemanal *ucavailability.DefinirGradeSemanalUseCase,
	consultarGradeSemanal *ucavailability.ConsultarGradeSemanalUseCase,
	criarExcecao *ucavailability.CriarExcecaoUseCase,
	removerExcecao *ucavailability.RemoverExcecaoUseCase,
	listarExcecoes *ucavailability.ListarExcecoesUseCase,
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool),
) *AvailabilityHandler {
	return &AvailabilityHandler{
		definirGradeSemanal:   definirGradeSemanal,
		consultarGradeSemanal: consultarGradeSemanal,
		criarExcecao:          criarExcecao,
		removerExcecao:        removerExcecao,
		listarExcecoes:        listarExcecoes,
		identidadeDoContexto:  identidadeDoContexto,
	}
}

// ConsultarGradeSemanal godoc
//
//	@Summary		Consultar grade semanal do prestador
//	@Description	Retorna a grade semanal de disponibilidade configurada pelo prestador autenticado
//	@Tags			availability
//	@Produce		json
//	@Success		200	{object}	dto.GradeSemanalResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Router			/providers/me/disponibilidade [get]
func (h *AvailabilityHandler) ConsultarGradeSemanal(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	out, err := h.consultarGradeSemanal.Executar(ucavailability.ConsultarGradeSemanalInput{ProviderID: id.UserID})
	if err != nil {
		responderErro(w, http.StatusInternalServerError, "erro interno")
		return
	}

	responderJSON(w, http.StatusOK, dto.GradeSemanalResponse{Dias: gradeParaDTO(out.Dias)})
}

// DefinirGradeSemanal godoc
//
//	@Summary		Definir grade semanal do prestador
//	@Description	Substitui por completo a grade semanal de disponibilidade do prestador autenticado
//	@Tags			availability
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.DefinirGradeSemanalRequest	true	"Grade semanal"
//	@Success		200		{object}	dto.GradeSemanalResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Router			/providers/me/disponibilidade [put]
func (h *AvailabilityHandler) DefinirGradeSemanal(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	var req dto.DefinirGradeSemanalRequest
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

	blocosPorDia := make(map[availability.DiaSemana][]ucavailability.BlocoInput, len(req.Dias))
	for _, dia := range req.Dias {
		blocos := make([]ucavailability.BlocoInput, 0, len(dia.Blocos))
		for _, b := range dia.Blocos {
			blocos = append(blocos, ucavailability.BlocoInput{InicioMinutos: b.InicioMinutos, FimMinutos: b.FimMinutos})
		}
		blocosPorDia[availability.DiaSemana(dia.DiaSemana)] = blocos
	}

	out, err := h.definirGradeSemanal.Executar(ucavailability.DefinirGradeSemanalInput{
		ProviderID:   id.UserID,
		BlocosPorDia: blocosPorDia,
	})
	if err != nil {
		responderErroDisponibilidade(w, err)
		return
	}

	responderJSON(w, http.StatusOK, dto.GradeSemanalResponse{Dias: gradeParaDTO(out.Dias)})
}

// ListarExcecoes godoc
//
//	@Summary		Listar exceções de data do prestador
//	@Description	Lista as exceções de data (bloqueios e extras) do prestador autenticado
//	@Tags			availability
//	@Produce		json
//	@Success		200	{object}	dto.ListarExcecoesResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Router			/providers/me/excecoes [get]
func (h *AvailabilityHandler) ListarExcecoes(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	out, err := h.listarExcecoes.Executar(ucavailability.ListarExcecoesInput{ProviderID: id.UserID})
	if err != nil {
		responderErro(w, http.StatusInternalServerError, "erro interno")
		return
	}

	respostas := make([]dto.ExcecaoResponse, 0, len(out.Excecoes))
	for _, e := range out.Excecoes {
		respostas = append(respostas, dto.ExcecaoResponse{
			ID:     e.ID,
			Data:   e.Data.Format(layoutData),
			Tipo:   string(e.Tipo),
			Blocos: blocosParaDTO(e.Blocos),
		})
	}

	responderJSON(w, http.StatusOK, dto.ListarExcecoesResponse{Excecoes: respostas})
}

// CriarExcecao godoc
//
//	@Summary		Criar exceção de data
//	@Description	Cria uma exceção de bloqueio ou extra para uma data específica do prestador autenticado
//	@Tags			availability
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.CriarExcecaoRequest	true	"Exceção de data"
//	@Success		201		{object}	dto.ExcecaoResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Router			/providers/me/excecoes [post]
func (h *AvailabilityHandler) CriarExcecao(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	var req dto.CriarExcecaoRequest
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

	data, err := time.Parse(layoutData, req.Data)
	if err != nil {
		responderErro(w, http.StatusBadRequest, "data inválida")
		return
	}

	blocos := make([]ucavailability.BlocoInput, 0, len(req.Blocos))
	for _, b := range req.Blocos {
		blocos = append(blocos, ucavailability.BlocoInput{InicioMinutos: b.InicioMinutos, FimMinutos: b.FimMinutos})
	}

	out, err := h.criarExcecao.Executar(ucavailability.CriarExcecaoInput{
		ProviderID: id.UserID,
		Data:       data,
		Tipo:       availability.TipoExcecao(req.Tipo),
		Blocos:     blocos,
	})
	if err != nil {
		responderErroDisponibilidade(w, err)
		return
	}

	responderJSON(w, http.StatusCreated, dto.ExcecaoResponse{
		ID:     out.ID,
		Data:   out.Data.Format(layoutData),
		Tipo:   string(out.Tipo),
		Blocos: blocosParaDTO(out.Blocos),
	})
}

// RemoverExcecao godoc
//
//	@Summary		Remover exceção de data
//	@Description	Remove uma exceção de data do prestador autenticado
//	@Tags			availability
//	@Success		204
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/providers/me/excecoes/{id} [delete]
func (h *AvailabilityHandler) RemoverExcecao(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	excecaoID := chi.URLParam(r, "id")
	err := h.removerExcecao.Executar(ucavailability.RemoverExcecaoInput{ProviderID: id.UserID, ExcecaoID: excecaoID})
	if err != nil {
		responderErroDisponibilidade(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func responderErroDisponibilidade(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, availability.ErrFimAntesDoInicio),
		errors.Is(err, availability.ErrForaDoDia),
		errors.Is(err, availability.ErrGranularidadeInvalida),
		errors.Is(err, availability.ErrBlocosSobrepostos),
		errors.Is(err, availability.ErrTipoInvalido),
		errors.Is(err, availability.ErrBloqueioComBlocos),
		errors.Is(err, availability.ErrExtraSemBlocos),
		errors.Is(err, availability.ErrProviderIDObrigatorio):
		responderErro(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ucavailability.ErrProviderNaoEncontrado):
		responderErro(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ucavailability.ErrExcecaoJaExiste):
		responderErro(w, http.StatusConflict, err.Error())
	case errors.Is(err, ucavailability.ErrExcecaoNaoEncontrada):
		responderErro(w, http.StatusNotFound, err.Error())
	default:
		responderErro(w, http.StatusInternalServerError, "erro interno")
	}
}

func gradeParaDTO(dias map[availability.DiaSemana][]availability.TimeBlock) []dto.DiaGradeDTO {
	resultado := make([]dto.DiaGradeDTO, 0, len(dias))
	for dia, blocos := range dias {
		resultado = append(resultado, dto.DiaGradeDTO{
			DiaSemana: int(dia),
			Blocos:    blocosParaDTO(blocos),
		})
	}
	return resultado
}

func blocosParaDTO(blocos []availability.TimeBlock) []dto.BlocoDTO {
	resultado := make([]dto.BlocoDTO, 0, len(blocos))
	for _, b := range blocos {
		resultado = append(resultado, dto.BlocoDTO{InicioMinutos: b.InicioMinutos, FimMinutos: b.FimMinutos})
	}
	return resultado
}
