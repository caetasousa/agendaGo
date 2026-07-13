package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"agendago/internal/adapter/http/dto"
	domappointment "agendago/internal/domain/appointment"
	domclient "agendago/internal/domain/client"
	ucappointment "agendago/internal/usecase/appointment"
	ucauth "agendago/internal/usecase/auth"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

// AppointmentHandler concentra os handlers do ciclo de vida do agendamento:
// consulta de slots livres, solicitação, transições de status e listagens.
type AppointmentHandler struct {
	consultarSlots       *ucappointment.ConsultarSlotsUseCase
	solicitar            *ucappointment.SolicitarUseCase
	solicitarConvidado   *ucappointment.SolicitarConvidadoUseCase
	transicionar         *ucappointment.TransicionarUseCase
	cancelarPorToken     *ucappointment.CancelarPorTokenUseCase
	listar               *ucappointment.ListarUseCase
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool)
}

// NovoAppointmentHandler cria uma instância de AppointmentHandler com os usecases injetados.
func NovoAppointmentHandler(
	consultarSlots *ucappointment.ConsultarSlotsUseCase,
	solicitar *ucappointment.SolicitarUseCase,
	solicitarConvidado *ucappointment.SolicitarConvidadoUseCase,
	transicionar *ucappointment.TransicionarUseCase,
	cancelarPorToken *ucappointment.CancelarPorTokenUseCase,
	listar *ucappointment.ListarUseCase,
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool),
) *AppointmentHandler {
	return &AppointmentHandler{
		consultarSlots:       consultarSlots,
		solicitar:            solicitar,
		solicitarConvidado:   solicitarConvidado,
		transicionar:         transicionar,
		cancelarPorToken:     cancelarPorToken,
		listar:               listar,
		identidadeDoContexto: identidadeDoContexto,
	}
}

// ConsultarSlots godoc
//
//	@Summary		Consultar horários livres de um prestador
//	@Description	Calcula os slots ofertáveis do período: disponibilidade do dia menos agendamentos, fatiada por duração + descanso
//	@Tags			appointments
//	@Produce		json
//	@Param			id	path		string	true	"ID do prestador"
//	@Param			de	query		string	true	"Data inicial (YYYY-MM-DD)"
//	@Param			ate	query		string	true	"Data final (YYYY-MM-DD)"
//	@Success		200	{object}	dto.SlotsResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/providers/{id}/slots [get]
func (h *AppointmentHandler) ConsultarSlots(w http.ResponseWriter, r *http.Request) {
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

	out, err := h.consultarSlots.Executar(ucappointment.ConsultarSlotsInput{
		ProviderID: chi.URLParam(r, "id"),
		De:         de,
		Ate:        ate,
		Agora:      time.Now(),
	})
	if err != nil {
		responderErroAgendamento(w, err)
		return
	}

	dias := make([]dto.DiaSlotsDTO, 0, len(out.Dias))
	for _, d := range out.Dias {
		slots := make([]dto.SlotDTO, 0, len(d.Slots))
		for _, s := range d.Slots {
			slots = append(slots, dto.SlotDTO{InicioMinutos: s.InicioMinutos, FimMinutos: s.FimMinutos})
		}
		dias = append(dias, dto.DiaSlotsDTO{Data: d.Data.Format(layoutData), Slots: slots})
	}

	responderJSON(w, http.StatusOK, dto.SlotsResponse{Dias: dias})
}

// Solicitar godoc
//
//	@Summary		Solicitar um agendamento
//	@Description	Reserva um slot livre para o cliente autenticado; a solicitação já ocupa o intervalo até o prestador responder ou o prazo expirar
//	@Tags			appointments
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.SolicitarAgendamentoRequest	true	"Slot desejado"
//	@Success		201		{object}	dto.AgendamentoResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Router			/agendamentos [post]
func (h *AppointmentHandler) Solicitar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	var req dto.SolicitarAgendamentoRequest
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

	out, err := h.solicitar.Executar(ucappointment.SolicitarInput{
		ClientID:      id.UserID,
		ProviderID:    req.ProviderID,
		Data:          data,
		InicioMinutos: req.InicioMinutos,
		Agora:         time.Now(),
	})
	if err != nil {
		responderErroAgendamento(w, err)
		return
	}

	responderJSON(w, http.StatusCreated, dto.AgendamentoResponse{
		ID:            out.ID,
		Data:          out.Data.Format(layoutData),
		InicioMinutos: out.InicioMinutos,
		FimMinutos:    out.FimMinutos,
		Status:        string(out.Status),
		ExpiraEm:      out.ExpiraEm.Format(time.RFC3339),
	})
}

// SolicitarConvidado godoc
//
//	@Summary		Solicitar um agendamento sem cadastro
//	@Description	Reserva um slot livre para um convidado (nome/email/telefone), sem exigir login. Rota pública.
//	@Tags			appointments
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.SolicitarConvidadoRequest	true	"Slot e contato do convidado"
//	@Success		201		{object}	dto.AgendamentoResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Router			/agendamentos/convidado [post]
func (h *AppointmentHandler) SolicitarConvidado(w http.ResponseWriter, r *http.Request) {
	var req dto.SolicitarConvidadoRequest
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

	out, err := h.solicitarConvidado.Executar(ucappointment.SolicitarConvidadoInput{
		ProviderID:    req.ProviderID,
		Data:          data,
		InicioMinutos: req.InicioMinutos,
		Nome:          req.Nome,
		Email:         req.Email,
		Telefone:      req.Telefone,
		Agora:         time.Now(),
	})
	if err != nil {
		responderErroAgendamento(w, err)
		return
	}

	responderJSON(w, http.StatusCreated, dto.AgendamentoResponse{
		ID:            out.ID,
		Data:          out.Data.Format(layoutData),
		InicioMinutos: out.InicioMinutos,
		FimMinutos:    out.FimMinutos,
		Status:        string(out.Status),
		ExpiraEm:      out.ExpiraEm.Format(time.RFC3339),
	})
}

// ListarDoPrestador godoc
//
//	@Summary		Listar agendamentos recebidos pelo prestador
//	@Description	Lista os agendamentos do prestador autenticado; solicitações vencidas viram EXPIRADO na leitura
//	@Tags			appointments
//	@Produce		json
//	@Success		200	{object}	dto.ListarAgendamentosResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Router			/providers/me/agendamentos [get]
func (h *AppointmentHandler) ListarDoPrestador(w http.ResponseWriter, r *http.Request) {
	// incluiContato=true: o prestador precisa do email/telefone para falar com
	// quem agendou, sobretudo convidados sem cadastro.
	h.listarAgendamentos(w, r, h.listar.DoPrestador, true)
}

// ListarDoCliente godoc
//
//	@Summary		Listar agendamentos do cliente
//	@Description	Lista os agendamentos feitos pelo cliente autenticado; solicitações vencidas viram EXPIRADO na leitura
//	@Tags			appointments
//	@Produce		json
//	@Success		200	{object}	dto.ListarAgendamentosResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Router			/clients/me/agendamentos [get]
func (h *AppointmentHandler) ListarDoCliente(w http.ResponseWriter, r *http.Request) {
	// incluiContato=false: o cliente não precisa ver o próprio contato repetido.
	h.listarAgendamentos(w, r, h.listar.DoCliente, false)
}

func (h *AppointmentHandler) listarAgendamentos(
	w http.ResponseWriter,
	r *http.Request,
	listar func(ucappointment.ListarInput) (*ucappointment.ListarOutput, error),
	incluiContato bool,
) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	out, err := listar(ucappointment.ListarInput{UsuarioID: id.UserID, Agora: time.Now()})
	if err != nil {
		responderErroAgendamento(w, err)
		return
	}

	agendamentos := make([]dto.AgendamentoResponse, 0, len(out.Agendamentos))
	for _, a := range out.Agendamentos {
		resp := dto.AgendamentoResponse{
			ID:            a.ID,
			Data:          a.Data.Format(layoutData),
			InicioMinutos: a.InicioMinutos,
			FimMinutos:    a.FimMinutos,
			Status:        string(a.Status),
			ExpiraEm:      a.ExpiraEm.Format(time.RFC3339),
			NomeCliente:   a.NomeCliente,
			NomePrestador: a.NomePrestador,
		}
		if incluiContato {
			resp.EmailCliente = a.EmailCliente
			resp.TelefoneCliente = a.TelefoneCliente
		}
		agendamentos = append(agendamentos, resp)
	}

	responderJSON(w, http.StatusOK, dto.ListarAgendamentosResponse{Agendamentos: agendamentos})
}

// Confirmar godoc
//
//	@Summary		Confirmar uma solicitação
//	@Description	O prestador aceita a solicitação pendente; o agendamento passa a CONFIRMADO
//	@Tags			appointments
//	@Success		204
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		409	{object}	map[string]string
//	@Router			/agendamentos/{id}/confirmar [post]
func (h *AppointmentHandler) Confirmar(w http.ResponseWriter, r *http.Request) {
	h.transicionarAgendamento(w, r, h.transicionar.Confirmar)
}

// Recusar godoc
//
//	@Summary		Recusar uma solicitação
//	@Description	O prestador nega a solicitação pendente; o intervalo volta a ficar livre
//	@Tags			appointments
//	@Success		204
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		409	{object}	map[string]string
//	@Router			/agendamentos/{id}/recusar [post]
func (h *AppointmentHandler) Recusar(w http.ResponseWriter, r *http.Request) {
	h.transicionarAgendamento(w, r, h.transicionar.Recusar)
}

// Cancelar godoc
//
//	@Summary		Cancelar um agendamento
//	@Description	Cliente ou prestador cancelam; confirmado exige antecedência mínima, solicitação pendente só pode ser cancelada pelo cliente
//	@Tags			appointments
//	@Success		204
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		409	{object}	map[string]string
//	@Router			/agendamentos/{id}/cancelar [post]
func (h *AppointmentHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	h.transicionarAgendamento(w, r, h.transicionar.Cancelar)
}

// DetalharCancelamento godoc
//
//	@Summary		Detalhar um agendamento por token de cancelamento
//	@Description	Rota pública: devolve os dados do agendamento apontado pelo token enviado ao convidado, para a página de cancelamento
//	@Tags			appointments
//	@Produce		json
//	@Param			token	path		string	true	"Token de cancelamento"
//	@Success		200		{object}	dto.DetalheCancelamentoResponse
//	@Failure		404		{object}	map[string]string
//	@Router			/agendamentos/cancelar/{token} [get]
func (h *AppointmentHandler) DetalharCancelamento(w http.ResponseWriter, r *http.Request) {
	out, err := h.cancelarPorToken.Detalhar(chi.URLParam(r, "token"), time.Now())
	if err != nil {
		responderErroAgendamento(w, err)
		return
	}

	responderJSON(w, http.StatusOK, dto.DetalheCancelamentoResponse{
		NomePrestador: out.NomePrestador,
		Data:          out.Data.Format(layoutData),
		InicioMinutos: out.InicioMinutos,
		FimMinutos:    out.FimMinutos,
		Status:        string(out.Status),
		PodeCancelar:  out.PodeCancelar,
	})
}

// CancelarPorToken godoc
//
//	@Summary		Cancelar um agendamento por token
//	@Description	Rota pública: o convidado cancela o agendamento pelo token do email. Respeita a antecedência mínima (24h para confirmados)
//	@Tags			appointments
//	@Param			token	path	string	true	"Token de cancelamento"
//	@Success		204
//	@Failure		404	{object}	map[string]string
//	@Failure		409	{object}	map[string]string
//	@Router			/agendamentos/cancelar/{token} [post]
func (h *AppointmentHandler) CancelarPorToken(w http.ResponseWriter, r *http.Request) {
	err := h.cancelarPorToken.Executar(chi.URLParam(r, "token"), time.Now())
	if err != nil {
		responderErroAgendamento(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MarcarRealizado godoc
//
//	@Summary		Marcar atendimento como realizado
//	@Description	O prestador conclui um agendamento confirmado cujo horário já passou
//	@Tags			appointments
//	@Success		204
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		409	{object}	map[string]string
//	@Router			/agendamentos/{id}/realizado [post]
func (h *AppointmentHandler) MarcarRealizado(w http.ResponseWriter, r *http.Request) {
	h.transicionarAgendamento(w, r, h.transicionar.MarcarRealizado)
}

// MarcarNaoCompareceu godoc
//
//	@Summary		Registrar não comparecimento
//	@Description	O prestador registra que o cliente não compareceu a um agendamento confirmado
//	@Tags			appointments
//	@Success		204
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		409	{object}	map[string]string
//	@Router			/agendamentos/{id}/nao-compareceu [post]
func (h *AppointmentHandler) MarcarNaoCompareceu(w http.ResponseWriter, r *http.Request) {
	h.transicionarAgendamento(w, r, h.transicionar.MarcarNaoCompareceu)
}

func (h *AppointmentHandler) transicionarAgendamento(
	w http.ResponseWriter,
	r *http.Request,
	transicao func(ucappointment.TransicionarInput) error,
) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	err := transicao(ucappointment.TransicionarInput{
		AgendamentoID: chi.URLParam(r, "id"),
		UsuarioID:     id.UserID,
		Tipo:          id.Tipo,
		Agora:         time.Now(),
	})
	if err != nil {
		responderErroAgendamento(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func responderErroAgendamento(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ucappointment.ErrPeriodoInvalido),
		errors.Is(err, domclient.ErrTelefoneObrigatorio),
		errors.Is(err, domclient.ErrNomeObrigatorio),
		errors.Is(err, domclient.ErrEmailObrigatorio):
		responderErro(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ucappointment.ErrProviderNaoEncontrado),
		errors.Is(err, ucappointment.ErrClientNaoEncontrado),
		errors.Is(err, ucappointment.ErrAgendamentoNaoEncontrado),
		errors.Is(err, ucappointment.ErrTokenCancelamentoInvalido):
		responderErro(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ucappointment.ErrClientInativo):
		responderErro(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ucappointment.ErrEmailTemConta):
		responderErro(w, http.StatusConflict, err.Error())
	case errors.Is(err, ucappointment.ErrHorarioIndisponivel),
		errors.Is(err, domappointment.ErrConflitoHorario),
		errors.Is(err, domappointment.ErrTransicaoInvalida),
		errors.Is(err, domappointment.ErrSolicitacaoExpirada),
		errors.Is(err, domappointment.ErrAntecedenciaInsuficiente),
		errors.Is(err, domappointment.ErrAtendimentoNaoIniciado):
		responderErro(w, http.StatusConflict, err.Error())
	default:
		responderErro(w, http.StatusInternalServerError, "erro interno")
	}
}
