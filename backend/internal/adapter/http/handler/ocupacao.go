package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"agendago/internal/adapter/http/dto"
	"agendago/internal/domain/availability"
	domocupacao "agendago/internal/domain/ocupacao"
	ucauth "agendago/internal/usecase/auth"
	ucocupacao "agendago/internal/usecase/ocupacao"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

// OcupacaoHandler concentra os handlers de compromisso pessoal do prestador.
// identidadeDoContexto é recebida como função para evitar import cycle entre
// os pacotes handler e middleware.
type OcupacaoHandler struct {
	criar                *ucocupacao.CriarUseCase
	listar               *ucocupacao.ListarUseCase
	remover              *ucocupacao.RemoverUseCase
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool)
}

// NovoOcupacaoHandler cria uma instância de OcupacaoHandler com os usecases injetados.
func NovoOcupacaoHandler(
	criar *ucocupacao.CriarUseCase,
	listar *ucocupacao.ListarUseCase,
	remover *ucocupacao.RemoverUseCase,
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool),
) *OcupacaoHandler {
	return &OcupacaoHandler{criar: criar, listar: listar, remover: remover, identidadeDoContexto: identidadeDoContexto}
}

// Listar godoc
//
//	@Summary		Listar compromissos pessoais do prestador
//	@Description	Compromissos do período (inclusivo) que deixam de ser ofertados na agenda
//	@Tags			ocupacoes
//	@Produce		json
//	@Param			de	query		string	true	"Data inicial (YYYY-MM-DD)"
//	@Param			ate	query		string	true	"Data final (YYYY-MM-DD)"
//	@Success		200	{object}	dto.OcupacoesResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Router			/providers/me/ocupacoes [get]
func (h *OcupacaoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	de, err := time.Parse(layoutData, r.URL.Query().Get("de"))
	if err != nil {
		responderErro(w, http.StatusBadRequest, "parâmetro 'de' inválido (use YYYY-MM-DD)")
		return
	}
	ate, err := time.Parse(layoutData, r.URL.Query().Get("ate"))
	if err != nil {
		responderErro(w, http.StatusBadRequest, "parâmetro 'ate' inválido (use YYYY-MM-DD)")
		return
	}

	ocupacoes, err := h.listar.Executar(ucocupacao.ListarInput{
		ProviderID: id.ProviderID, De: de, Ate: ate,
	})
	if err != nil {
		responderErroOcupacao(w, r, err)
		return
	}

	resp := dto.OcupacoesResponse{Ocupacoes: make([]dto.OcupacaoResponse, 0, len(ocupacoes))}
	for _, o := range ocupacoes {
		resp.Ocupacoes = append(resp.Ocupacoes, paraOcupacaoResponse(o))
	}
	responderJSON(w, http.StatusOK, resp)
}

// Criar godoc
//
//	@Summary		Registrar compromisso pessoal
//	@Description	Reserva um intervalo do dia para o prestador; o horário some da oferta sem redefinir o expediente
//	@Tags			ocupacoes
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CriarOcupacaoRequest	true	"Compromisso"
//	@Success		201		{object}	dto.OcupacaoResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Router			/providers/me/ocupacoes [post]
func (h *OcupacaoHandler) Criar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	var req dto.CriarOcupacaoRequest
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
		responderErro(w, http.StatusBadRequest, "data inválida (use YYYY-MM-DD)")
		return
	}

	o, err := h.criar.Executar(ucocupacao.CriarInput{
		ProviderID:    id.ProviderID,
		Data:          data,
		InicioMinutos: req.InicioMinutos,
		FimMinutos:    req.FimMinutos,
		Titulo:        req.Titulo,
		Agora:         time.Now(),
	})
	if err != nil {
		responderErroOcupacao(w, r, err)
		return
	}
	responderJSON(w, http.StatusCreated, paraOcupacaoResponse(o))
}

// Remover godoc
//
//	@Summary		Remover compromisso pessoal
//	@Description	Devolve o intervalo à oferta de horários
//	@Tags			ocupacoes
//	@Param			id	path	string	true	"ID do compromisso"
//	@Success		204	"sem conteúdo"
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/providers/me/ocupacoes/{id} [delete]
func (h *OcupacaoHandler) Remover(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	err := h.remover.Executar(ucocupacao.RemoverInput{
		ID: chi.URLParam(r, "id"), ProviderID: id.ProviderID,
	})
	if err != nil {
		responderErroOcupacao(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func paraOcupacaoResponse(o *domocupacao.Ocupacao) dto.OcupacaoResponse {
	return dto.OcupacaoResponse{
		ID:            o.ID,
		Data:          o.Data.Format(layoutData),
		InicioMinutos: o.InicioMinutos,
		FimMinutos:    o.FimMinutos,
		Titulo:        o.Titulo,
		Origem:        string(o.Origem),
	}
}

// responderErroOcupacao traduz os erros de compromisso em status HTTP, no
// mesmo formato de responderErroDisponibilidade.
//
// O conflito com agendamento é 409, não 400: o pedido está correto, o estado
// atual é que não permite. Quem decide o que fazer é o prestador — cancelar o
// cliente primeiro —, e o sistema não desmarca ninguém por conta própria.
//
// As regras de intervalo vêm de availability, porque o compromisso reusa
// NovoTimeBlock: um intervalo do dia é um intervalo do dia.
func responderErroOcupacao(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domocupacao.ErrConflitoComAgendamento):
		responderErro(w, http.StatusConflict, err.Error())
	case errors.Is(err, ucocupacao.ErrOcupacaoNaoEncontrada):
		responderErro(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domocupacao.ErrProviderObrigatorio),
		errors.Is(err, domocupacao.ErrOrigemInvalida),
		errors.Is(err, domocupacao.ErrTituloLongo),
		errors.Is(err, ucocupacao.ErrPeriodoInvalido),
		errors.Is(err, availability.ErrFimAntesDoInicio),
		errors.Is(err, availability.ErrForaDoDia),
		errors.Is(err, availability.ErrGranularidadeInvalida):
		responderErro(w, http.StatusBadRequest, err.Error())
	default:
		responderErroInterno(w, r, err)
	}
}
