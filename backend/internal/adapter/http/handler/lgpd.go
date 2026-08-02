package handler

import (
	"errors"
	"net/http"

	"agendago/internal/adapter/http/dto"
	ucauth "agendago/internal/usecase/auth"
	"agendago/internal/usecase/lgpd"
)

// LgpdHandler atende os pedidos do titular sobre os próprios dados.
// identidadeDoContexto é recebida como função para evitar import cycle entre
// os pacotes handler e middleware.
type LgpdHandler struct {
	exportar             *lgpd.ExportarUseCase
	anonimizar           *lgpd.AnonimizarUseCase
	cookieSeguro         bool
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool)
}

// NovoLgpdHandler cria uma instância de LgpdHandler com os usecases injetados.
func NovoLgpdHandler(
	exportar *lgpd.ExportarUseCase,
	anonimizar *lgpd.AnonimizarUseCase,
	cookieSeguro bool,
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool),
) *LgpdHandler {
	return &LgpdHandler{exportar: exportar, anonimizar: anonimizar, cookieSeguro: cookieSeguro, identidadeDoContexto: identidadeDoContexto}
}

// ExportarDados godoc
//
//	@Summary		Exportar os próprios dados
//	@Description	Devolve cadastro e histórico de agendamentos do cliente autenticado, para download
//	@Tags			lgpd
//	@Produce		json
//	@Success		200	{object}	dto.DadosExportadosResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Router			/clients/me/dados [get]
func (h *LgpdHandler) ExportarDados(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	dados, err := h.exportar.Executar(id.UserID)
	if err != nil {
		if errors.Is(err, lgpd.ErrClienteNaoEncontrado) {
			responderErro(w, http.StatusNotFound, err.Error())
			return
		}
		responderErroInterno(w, r, err)
		return
	}

	resp := dto.DadosExportadosResponse{
		ID:           dados.ID,
		Nome:         dados.Nome,
		Email:        dados.Email,
		Telefone:     dados.Telefone,
		CriadoEm:     dados.CriadoEm.Format("2006-01-02T15:04:05Z07:00"),
		Agendamentos: make([]dto.AgendamentoExportadoDTO, 0, len(dados.Agendamentos)),
		Total:        dados.TotalNoPeriodo,
		Truncado:     dados.Truncado,
	}
	for _, a := range dados.Agendamentos {
		resp.Agendamentos = append(resp.Agendamentos, dto.AgendamentoExportadoDTO{
			Data:          a.Data.Format(layoutData),
			InicioMinutos: a.InicioMinutos,
			FimMinutos:    a.FimMinutos,
			Status:        a.Status,
			CriadoEm:      a.CriadoEm.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	// Content-Disposition faz o navegador baixar em vez de renderizar — é um
	// pacote de dados para guardar, não uma página.
	w.Header().Set("Content-Disposition", `attachment; filename="meus-dados-agendago.json"`)
	responderJSON(w, http.StatusOK, resp)
}

// RemoverConta godoc
//
//	@Summary		Remover a própria conta
//	@Description	Anonimiza o cadastro e encerra as sessões; os agendamentos permanecem na agenda do prestador sem identificar o cliente
//	@Tags			lgpd
//	@Success		204	"sem conteúdo"
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		409	{object}	map[string]string
//	@Router			/clients/me [delete]
func (h *LgpdHandler) RemoverConta(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	err := h.anonimizar.Executar(id.UserID)
	switch {
	case err == nil:
		// O cookie morre junto: a sessão já foi revogada no servidor, e deixar
		// o cookie no navegador só produziria 401 confusos na próxima página.
		http.SetCookie(w, cookieSessaoExpirado(h.cookieSeguro))
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, lgpd.ErrClienteNaoEncontrado):
		responderErro(w, http.StatusNotFound, err.Error())
	case errors.Is(err, lgpd.ErrJaAnonimizado):
		responderErro(w, http.StatusConflict, err.Error())
	default:
		responderErroInterno(w, r, err)
	}
}
