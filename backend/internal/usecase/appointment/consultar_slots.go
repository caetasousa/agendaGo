package appointment

import (
	"time"

	domavailability "agendago/internal/domain/availability"
	"agendago/internal/domain/slot"
	ucavailability "agendago/internal/usecase/availability"
)

// maxDiasSlots limita o período consultável de uma vez, como na agenda.
const maxDiasSlots = 92

// resolvedorDisponibilidade resolve os blocos efetivos de uma data —
// implementado por availability.ConsultarDisponibilidadeUseCase.
type resolvedorDisponibilidade interface {
	Executar(in ucavailability.ConsultarDisponibilidadeInput) ([]domavailability.TimeBlock, error)
}

// ConsultarSlotsInput define o prestador, o período (inclusivo) e o instante
// da consulta — Agora vem do chamador para a regra ser testável.
type ConsultarSlotsInput struct {
	ProviderID string
	De         time.Time
	Ate        time.Time
	Agora      time.Time
	// IncluirAgendaFechada oferta os slots mesmo com AceitaAgendamentos
	// desligado — só para o próprio prestador marcando na sua agenda (a
	// identidade vem da sessão, nunca de entrada pública). Agenda fechada
	// esconde os horários do público, não do dono.
	IncluirAgendaFechada bool
}

// DiaSlots são os horários livres ofertáveis de uma data.
type DiaSlots struct {
	Data  time.Time
	Slots []slot.Slot
}

// ConsultarSlotsOutput contém os slots livres do período, dia a dia.
type ConsultarSlotsOutput struct {
	Dias []DiaSlots
}

// ConsultarSlotsUseCase calcula os horários ofertáveis de um prestador:
// disponibilidade resolvida do dia, menos agendamentos que ocupam horário,
// fatiada pela duração do atendimento + descanso do prestador.
type ConsultarSlotsUseCase struct {
	resolvedor      resolvedorDisponibilidade
	appointmentRepo repositorioAppointment
	providerRepo    repositorioProvider
	fuso            *time.Location
}

// NovoConsultarSlotsUseCase cria uma instância de ConsultarSlotsUseCase com as dependências injetadas.
func NovoConsultarSlotsUseCase(
	resolvedor resolvedorDisponibilidade,
	appointmentRepo repositorioAppointment,
	providerRepo repositorioProvider,
	fuso *time.Location,
) *ConsultarSlotsUseCase {
	return &ConsultarSlotsUseCase{
		resolvedor:      resolvedor,
		appointmentRepo: appointmentRepo,
		providerRepo:    providerRepo,
		fuso:            fuso,
	}
}

// Executar resolve os slots livres de cada dia do período. Dias passados (e
// horários de hoje que já começaram) nunca são ofertados; prestador com a
// agenda desativada não oferta slot algum. Retorna ErrPeriodoInvalido para
// período invertido ou maior que maxDiasSlots.
func (uc *ConsultarSlotsUseCase) Executar(in ConsultarSlotsInput) (*ConsultarSlotsOutput, error) {
	if in.Ate.Before(in.De) || in.Ate.Sub(in.De) > maxDiasSlots*24*time.Hour {
		return nil, ErrPeriodoInvalido
	}

	p, err := uc.providerRepo.BuscarPorID(in.ProviderID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProviderNaoEncontrado
	}

	ocupantes, err := uc.appointmentRepo.ListarOcupantesPorPeriodo(in.ProviderID, in.De, in.Ate, in.Agora)
	if err != nil {
		return nil, err
	}
	ocupadosPorDia := make(map[string][]slot.Intervalo)
	for _, a := range ocupantes {
		chave := a.Data.Format("2006-01-02")
		ocupadosPorDia[chave] = append(ocupadosPorDia[chave], slot.Intervalo{
			InicioMinutos: a.InicioMinutos,
			FimMinutos:    a.FimMinutos,
		})
	}

	agoraLocal := in.Agora.In(uc.fuso)
	hoje := time.Date(agoraLocal.Year(), agoraLocal.Month(), agoraLocal.Day(), 0, 0, 0, 0, time.UTC)
	minutosAgora := agoraLocal.Hour()*60 + agoraLocal.Minute()

	var dias []DiaSlots
	for d := in.De; !d.After(in.Ate); d = d.AddDate(0, 0, 1) {
		dia := DiaSlots{Data: d}
		diaTruncado := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)

		if p.Ativo && (p.AceitaAgendamentos || in.IncluirAgendaFechada) && !diaTruncado.Before(hoje) {
			blocos, err := uc.resolvedor.Executar(ucavailability.ConsultarDisponibilidadeInput{
				ProviderID:           in.ProviderID,
				Data:                 d,
				IncluirAgendaFechada: in.IncluirAgendaFechada,
			})
			if err != nil {
				return nil, err
			}

			livres := slot.Livres(blocos, ocupadosPorDia[d.Format("2006-01-02")], p.DuracaoAtendimentoMinutos, p.DescansoMinutos)
			if diaTruncado.Equal(hoje) {
				livres = somenteFuturos(livres, minutosAgora)
			}
			dia.Slots = livres
		}

		dias = append(dias, dia)
	}

	return &ConsultarSlotsOutput{Dias: dias}, nil
}

// somenteFuturos descarta os slots de hoje que já começaram — não se oferta
// horário no passado.
func somenteFuturos(slots []slot.Slot, minutosAgora int) []slot.Slot {
	var futuros []slot.Slot
	for _, s := range slots {
		if s.InicioMinutos > minutosAgora {
			futuros = append(futuros, s)
		}
	}
	return futuros
}
