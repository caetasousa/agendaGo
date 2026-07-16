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
// (agenda resolvida e definições por data). identidadeDoContexto extrai a
// identidade posta no contexto pelo middleware de autenticação — recebida
// como função para evitar um import cycle entre os pacotes handler e middleware.
type AvailabilityHandler struct {
	consultarAgenda      *ucavailability.ConsultarAgendaUseCase
	definirDia           *ucavailability.DefinirDiaUseCase
	removerDia           *ucavailability.RemoverDiaUseCase
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool)
}

// NovoAvailabilityHandler cria uma instância de AvailabilityHandler com os usecases injetados.
func NovoAvailabilityHandler(
	consultarAgenda *ucavailability.ConsultarAgendaUseCase,
	definirDia *ucavailability.DefinirDiaUseCase,
	removerDia *ucavailability.RemoverDiaUseCase,
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool),
) *AvailabilityHandler {
	return &AvailabilityHandler{
		consultarAgenda:      consultarAgenda,
		definirDia:           definirDia,
		removerDia:           removerDia,
		identidadeDoContexto: identidadeDoContexto,
	}
}

// ConsultarAgenda godoc
//
//	@Summary		Consultar agenda resolvida do prestador
//	@Description	Resolve a disponibilidade de cada dia do período (inclusivo): definição própria da data ou expediente padrão
//	@Tags			availability
//	@Produce		json
//	@Param			de	query		string	true	"Data inicial (YYYY-MM-DD)"
//	@Param			ate	query		string	true	"Data final (YYYY-MM-DD)"
//	@Success		200	{object}	dto.AgendaResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Router			/providers/me/agenda [get]
func (h *AvailabilityHandler) ConsultarAgenda(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	de, err := time.Parse(layoutData, r.URL.Query().Get("de"))
	if err != nil {
		responderErro(w, http.StatusBadRequest, "parâmetro 'de' inválido (YYYY-MM-DD)")
		return
	}
	ate, err := time.Parse(layoutData, r.URL.Query().Get("ate"))
	if err != nil {
		responderErro(w, http.StatusBadRequest, "parâmetro 'ate' inválido (YYYY-MM-DD)")
		return
	}

	out, err := h.consultarAgenda.Executar(ucavailability.ConsultarAgendaInput{
		ProviderID: id.UserID,
		De:         de,
		Ate:        ate,
	})
	if err != nil {
		responderErroDisponibilidade(w, r, err)
		return
	}

	dias := make([]dto.DiaAgendaDTO, 0, len(out.Dias))
	for _, d := range out.Dias {
		dias = append(dias, dto.DiaAgendaDTO{
			Data:   d.Data.Format(layoutData),
			Origem: string(d.Origem),
			Blocos: blocosParaDTO(d.Blocos),
		})
	}

	responderJSON(w, http.StatusOK, dto.AgendaResponse{
		AceitaAgendamentos: out.AceitaAgendamentos,
		Dias:               dias,
	})
}

// DefinirDia godoc
//
//	@Summary		Definir um dia específico
//	@Description	Cria ou substitui a definição própria da data: bloqueio (dia indisponível) ou extra (horários personalizados)
//	@Tags			availability
//	@Accept			json
//	@Produce		json
//	@Param			data	path		string					true	"Data (YYYY-MM-DD)"
//	@Param			body	body		dto.DefinirDiaRequest	true	"Definição do dia"
//	@Success		200		{object}	dto.DiaAgendaDTO
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Router			/providers/me/dias/{data} [put]
func (h *AvailabilityHandler) DefinirDia(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	data, err := time.Parse(layoutData, chi.URLParam(r, "data"))
	if err != nil {
		responderErro(w, http.StatusBadRequest, "data inválida (YYYY-MM-DD)")
		return
	}

	var req dto.DefinirDiaRequest
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

	blocos := make([]ucavailability.BlocoInput, 0, len(req.Blocos))
	for _, b := range req.Blocos {
		blocos = append(blocos, ucavailability.BlocoInput{InicioMinutos: b.InicioMinutos, FimMinutos: b.FimMinutos})
	}

	out, err := h.definirDia.Executar(ucavailability.DefinirDiaInput{
		ProviderID: id.UserID,
		Data:       data,
		Tipo:       availability.TipoExcecao(req.Tipo),
		Blocos:     blocos,
	})
	if err != nil {
		responderErroDisponibilidade(w, r, err)
		return
	}

	responderJSON(w, http.StatusOK, dto.DiaAgendaDTO{
		Data:   out.Data.Format(layoutData),
		Origem: string(out.Tipo),
		Blocos: blocosParaDTO(out.Blocos),
	})
}

// RemoverDia godoc
//
//	@Summary		Remover a definição de um dia
//	@Description	Remove a definição própria da data; o dia volta ao expediente padrão do prestador
//	@Tags			availability
//	@Success		204
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/providers/me/dias/{data} [delete]
func (h *AvailabilityHandler) RemoverDia(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	data, err := time.Parse(layoutData, chi.URLParam(r, "data"))
	if err != nil {
		responderErro(w, http.StatusBadRequest, "data inválida (YYYY-MM-DD)")
		return
	}

	if err := h.removerDia.Executar(ucavailability.RemoverDiaInput{ProviderID: id.UserID, Data: data}); err != nil {
		responderErroDisponibilidade(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func responderErroDisponibilidade(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, availability.ErrFimAntesDoInicio),
		errors.Is(err, availability.ErrForaDoDia),
		errors.Is(err, availability.ErrGranularidadeInvalida),
		errors.Is(err, availability.ErrBlocosSobrepostos),
		errors.Is(err, availability.ErrTipoInvalido),
		errors.Is(err, availability.ErrBloqueioComBlocos),
		errors.Is(err, availability.ErrExtraSemBlocos),
		errors.Is(err, availability.ErrProviderIDObrigatorio),
		errors.Is(err, ucavailability.ErrPeriodoInvalido):
		responderErro(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ucavailability.ErrProviderNaoEncontrado):
		responderErro(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ucavailability.ErrDiaNaoDefinido):
		responderErro(w, http.StatusNotFound, err.Error())
	default:
		responderErroInterno(w, r, err)
	}
}

func blocosParaDTO(blocos []availability.TimeBlock) []dto.BlocoDTO {
	resultado := make([]dto.BlocoDTO, 0, len(blocos))
	for _, b := range blocos {
		resultado = append(resultado, dto.BlocoDTO{InicioMinutos: b.InicioMinutos, FimMinutos: b.FimMinutos})
	}
	return resultado
}
