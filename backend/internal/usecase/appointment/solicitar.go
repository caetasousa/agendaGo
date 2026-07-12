package appointment

import (
	"time"

	"agendago/internal/domain/appointment"

	"github.com/google/uuid"
)

// SolicitarInput contém os dados da solicitação. ClientID vem da identidade
// da sessão autenticada, nunca do corpo da requisição.
type SolicitarInput struct {
	ClientID      string
	ProviderID    string
	Data          time.Time
	InicioMinutos int
	Agora         time.Time
}

// SolicitarOutput contém a solicitação criada, já ocupando o intervalo.
type SolicitarOutput struct {
	ID            string
	ProviderID    string
	Data          time.Time
	InicioMinutos int
	FimMinutos    int
	Status        appointment.Status
	ExpiraEm      time.Time
}

// SolicitarUseCase cria a solicitação de agendamento: valida que o horário é
// um slot livre de verdade e persiste com reserva pessimista — a solicitação
// já ocupa o intervalo, com expiração em agora+TTL.
type SolicitarUseCase struct {
	consultarSlots *ConsultarSlotsUseCase
	repo           repositorioAppointment
	clientRepo     repositorioClient
	ttl            time.Duration
}

// NovoSolicitarUseCase cria uma instância de SolicitarUseCase com as dependências injetadas.
func NovoSolicitarUseCase(
	consultarSlots *ConsultarSlotsUseCase,
	repo repositorioAppointment,
	clientRepo repositorioClient,
	ttl time.Duration,
) *SolicitarUseCase {
	return &SolicitarUseCase{consultarSlots: consultarSlots, repo: repo, clientRepo: clientRepo, ttl: ttl}
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

	return uc.reservar(in.ProviderID, in.ClientID, in.Data, in.InicioMinutos, in.Agora)
}

// reservar valida que InicioMinutos é um slot livre de verdade (via
// ConsultarSlots) e persiste a solicitação sob a barreira anti-overbooking.
// Compartilhado pela solicitação autenticada e pela de convidado.
func (uc *SolicitarUseCase) reservar(providerID, clientID string, data time.Time, inicioMinutos int, agora time.Time) (*SolicitarOutput, error) {
	slots, err := uc.consultarSlots.Executar(ConsultarSlotsInput{
		ProviderID: providerID,
		De:         data,
		Ate:        data,
		Agora:      agora,
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

	if err := uc.repo.SalvarSeLivre(novo, agora); err != nil {
		return nil, err
	}

	return &SolicitarOutput{
		ID:            novo.ID,
		ProviderID:    novo.ProviderID,
		Data:          novo.Data,
		InicioMinutos: novo.InicioMinutos,
		FimMinutos:    novo.FimMinutos,
		Status:        novo.Status,
		ExpiraEm:      novo.ExpiraEm,
	}, nil
}
