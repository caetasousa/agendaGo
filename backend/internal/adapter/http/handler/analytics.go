package handler

import (
	"errors"
	"net/http"
	"time"

	"agendago/internal/adapter/http/dto"
	ucanalytics "agendago/internal/usecase/analytics"
	ucauth "agendago/internal/usecase/auth"
	ucavailability "agendago/internal/usecase/availability"
)

// AnalyticsHandler expõe os números da agenda do prestador autenticado.
// identidadeDoContexto extrai a identidade posta no contexto pelo middleware
// de autenticação — recebida como função para evitar um import cycle entre os
// pacotes handler e middleware.
type AnalyticsHandler struct {
	metricas             *ucanalytics.MetricasUseCase
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool)
}

// NovoAnalyticsHandler cria uma instância de AnalyticsHandler com o usecase injetado.
func NovoAnalyticsHandler(
	metricas *ucanalytics.MetricasUseCase,
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool),
) *AnalyticsHandler {
	return &AnalyticsHandler{metricas: metricas, identidadeDoContexto: identidadeDoContexto}
}

// Metricas godoc
//
//	@Summary		Métricas da agenda do prestador
//	@Description	Resume o período (recortado pela data do atendimento): quantos agendamentos em cada status, quanto do expediente ofertado foi reservado e a taxa de comparecimento
//	@Tags			analytics
//	@Produce		json
//	@Param			de	query		string	true	"Data inicial (YYYY-MM-DD)"
//	@Param			ate	query		string	true	"Data final (YYYY-MM-DD)"
//	@Success		200	{object}	dto.MetricasResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/providers/me/metricas [get]
func (h *AnalyticsHandler) Metricas(w http.ResponseWriter, r *http.Request) {
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

	out, err := h.metricas.Executar(ucanalytics.MetricasInput{
		ProviderID: id.ProviderID,
		De:         de,
		Ate:        ate,
		Agora:      time.Now(),
	})
	if err != nil {
		responderErroMetricas(w, r, err)
		return
	}

	porStatus := make(map[string]int, len(out.PorStatus))
	for status, quantidade := range out.PorStatus {
		porStatus[string(status)] = quantidade
	}

	responderJSON(w, http.StatusOK, dto.MetricasResponse{
		De:                 out.De.Format(layoutData),
		Ate:                out.Ate.Format(layoutData),
		PorStatus:          porStatus,
		Total:              out.Total,
		MinutosOfertados:   out.MinutosOfertados,
		MinutosReservados:  out.MinutosReservados,
		TaxaOcupacao:       out.TaxaOcupacao,
		TaxaComparecimento: out.TaxaComparecimento,
	})
}

func responderErroMetricas(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ucanalytics.ErrPeriodoInvalido):
		responderErro(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ucavailability.ErrProviderNaoEncontrado):
		responderErro(w, http.StatusNotFound, err.Error())
	default:
		responderErroInterno(w, r, err)
	}
}
