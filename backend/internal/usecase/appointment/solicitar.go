package appointment

import (
	"time"

	"agendago/internal/domain/appointment"

	"github.com/google/uuid"
)

// SolicitarInput contém os dados da solicitação. ClientID vem da identidade
// da sessão autenticada, nunca do corpo da requisição. Observacao é uma nota
// livre e opcional, visível ao prestador na lista de agendamentos.
type SolicitarInput struct {
	ClientID      string
	ProviderID    string
	Data          time.Time
	InicioMinutos int
	Agora         time.Time
	Observacao    string
}

// SolicitarOutput contém a solicitação criada, já ocupando o intervalo.
type SolicitarOutput struct {
	ID                   string
	ProviderID           string
	Data                 time.Time
	InicioMinutos        int
	FimMinutos           int
	Status               appointment.Status
	ExpiraEm             time.Time
	Observacao           string
	MarcadoPeloPrestador bool
}

// SolicitarUseCase cria a solicitação de agendamento: valida que o horário é
// um slot livre de verdade e persiste com reserva pessimista — a solicitação
// já ocupa o intervalo, com expiração em agora+TTL.
type SolicitarUseCase struct {
	consultarSlots *ConsultarSlotsUseCase
	repo           repositorioAppointment
	clientRepo     repositorioClient
	providerRepo   repositorioProvider
	notificador    notificadorAgendamento
	ttl            time.Duration
}

// NovoSolicitarUseCase cria uma instância de SolicitarUseCase com as dependências injetadas.
func NovoSolicitarUseCase(
	consultarSlots *ConsultarSlotsUseCase,
	repo repositorioAppointment,
	clientRepo repositorioClient,
	providerRepo repositorioProvider,
	notificador notificadorAgendamento,
	ttl time.Duration,
) *SolicitarUseCase {
	return &SolicitarUseCase{
		consultarSlots: consultarSlots,
		repo:           repo,
		clientRepo:     clientRepo,
		providerRepo:   providerRepo,
		notificador:    notificador,
		ttl:            ttl,
	}
}

// Executar valida o cliente e o slot (disponibilidade do dia, ocupação,
// horário futuro, agenda ativa — tudo via ConsultarSlots) e persiste a
// solicitação sob a barreira anti-overbooking. Retorna ErrHorarioIndisponivel
// quando o horário não é ofertável, inclusive na corrida entre a checagem e a
// escrita.
func (uc *SolicitarUseCase) Executar(in SolicitarInput) (*SolicitarOutput, error) {
	c, err := uc.clientRepo.BuscarPorID(in.ClientID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrClientNaoEncontrado
	}
	// a sessão pode ter nascido antes do banimento — re-checa o status aqui
	if !c.Ativo {
		return nil, ErrClientInativo
	}

	return uc.reservar(in.ProviderID, in.ClientID, in.Data, in.InicioMinutos, in.Agora, in.Observacao)
}

// reservar valida que InicioMinutos é um slot livre de verdade (via
// ConsultarSlots) e persiste a solicitação sob a barreira anti-overbooking,
// notificando o prestador do novo pedido. Compartilhado pela solicitação
// autenticada e pela de convidado.
func (uc *SolicitarUseCase) reservar(providerID, clientID string, data time.Time, inicioMinutos int, agora time.Time, observacao string) (*SolicitarOutput, error) {
	novo, err := uc.reservarSlot(providerID, clientID, data, inicioMinutos, agora, false, observacao, false)
	if err != nil {
		return nil, err
	}
	uc.notificarSolicitacao(novo)
	return novaSaidaSolicitar(novo), nil
}

// reservarSlot é o núcleo da reserva, sem notificação: valida o slot, cria a
// solicitação e persiste sob a barreira anti-overbooking. O
// incluirAgendaFechada só é usado quando o próprio prestador marca na sua
// agenda — o público nunca enxerga slots de agenda fechada. Quando
// comoRegistroDoPrestador é true, o agendamento já nasce CONFIRMADO e
// marcado como registro do prestador (ver MarcarComoRegistroDoPrestador).
func (uc *SolicitarUseCase) reservarSlot(providerID, clientID string, data time.Time, inicioMinutos int, agora time.Time, incluirAgendaFechada bool, observacao string, comoRegistroDoPrestador bool) (*appointment.Appointment, error) {
	slots, err := uc.consultarSlots.Executar(ConsultarSlotsInput{
		ProviderID:           providerID,
		De:                   data,
		Ate:                  data,
		Agora:                agora,
		IncluirAgendaFechada: incluirAgendaFechada,
	})
	if err != nil {
		return nil, err
	}

	fimMinutos := -1
	for _, dia := range slots.Dias {
		for _, s := range dia.Slots {
			if s.InicioMinutos == inicioMinutos {
				fimMinutos = s.FimMinutos
			}
		}
	}
	if fimMinutos < 0 {
		return nil, ErrHorarioIndisponivel
	}

	novo, err := appointment.Novo(uuid.NewString(), providerID, clientID, data, inicioMinutos, fimMinutos, agora, uc.ttl)
	if err != nil {
		return nil, err
	}
	novo.Observacao = observacao
	if comoRegistroDoPrestador {
		novo.MarcarComoRegistroDoPrestador(agora)
	}

	if err := uc.repo.SalvarSeLivre(novo, agora); err != nil {
		return nil, err
	}

	return novo, nil
}

// novaSaidaSolicitar projeta o agendamento persistido na saída dos usecases
// de solicitação.
func novaSaidaSolicitar(a *appointment.Appointment) *SolicitarOutput {
	return &SolicitarOutput{
		ID:                   a.ID,
		ProviderID:           a.ProviderID,
		Data:                 a.Data,
		InicioMinutos:        a.InicioMinutos,
		FimMinutos:           a.FimMinutos,
		Status:               a.Status,
		ExpiraEm:             a.ExpiraEm,
		Observacao:           a.Observacao,
		MarcadoPeloPrestador: a.MarcadoPeloPrestador,
	}
}

// notificarSolicitacao avisa o prestador do novo pedido. Best-effort: se não
// conseguir resolver os nomes/emails das partes, a notificação é só
// silenciosamente pulada — nunca falha a reserva já persistida.
func (uc *SolicitarUseCase) notificarSolicitacao(a *appointment.Appointment) {
	p, err := uc.providerRepo.BuscarPorID(a.ProviderID)
	if err != nil || p == nil {
		return
	}
	c, err := uc.clientRepo.BuscarPorID(a.ClientID)
	if err != nil || c == nil {
		return
	}

	uc.notificador.NotificarSolicitacao(NotificacaoAgendamento{
		NomePrestador:  p.Nome,
		EmailPrestador: p.Email,
		NomeCliente:    c.Nome,
		EmailCliente:   c.Email,
		Data:           a.Data,
		InicioMinutos:  a.InicioMinutos,
		FimMinutos:     a.FimMinutos,
		ExpiraEm:       a.ExpiraEm,
	})
}
