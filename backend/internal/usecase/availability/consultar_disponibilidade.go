package availability

import (
	"time"

	"agendago/internal/domain/availability"
)

// blocosComerciaisPadrao é o default de "dia comercial" quando o prestador
// aceita agendamentos mas nunca configurou a grade semanal: 08:00–12:00 e
// 14:00–18:00, de segunda a sexta.
var blocosComerciaisPadrao = []availability.TimeBlock{
	{InicioMinutos: 8 * 60, FimMinutos: 12 * 60},
	{InicioMinutos: 14 * 60, FimMinutos: 18 * 60},
}

// ConsultarDisponibilidadeInput define o prestador e a data (já no fuso
// correto, resolvida pelo chamador) a consultar.
type ConsultarDisponibilidadeInput struct {
	ProviderID string
	Data       time.Time
}

// ConsultarDisponibilidadeUseCase resolve os blocos de horário efetivamente
// disponíveis de um prestador em uma data — reutilizável pelo futuro domínio
// de Slots. Não conhece HTTP nem cálculo de slot.
type ConsultarDisponibilidadeUseCase struct {
	scheduleRepo repositorioWeeklySchedule
	excecaoRepo  repositorioDateException
	providerRepo repositorioProvider
}

// NovoConsultarDisponibilidadeUseCase cria uma instância de ConsultarDisponibilidadeUseCase com os repositórios injetados.
func NovoConsultarDisponibilidadeUseCase(
	scheduleRepo repositorioWeeklySchedule,
	excecaoRepo repositorioDateException,
	providerRepo repositorioProvider,
) *ConsultarDisponibilidadeUseCase {
	return &ConsultarDisponibilidadeUseCase{scheduleRepo: scheduleRepo, excecaoRepo: excecaoRepo, providerRepo: providerRepo}
}

// Executar resolve os blocos efetivos de ProviderID em Data seguindo a ordem:
// exceção da data → padrão semanal → (se AceitaAgendamentos e nada
// configurado) default comercial. Retorna slice vazia (nunca nil-erro) quando
// o prestador não atende no dia.
func (uc *ConsultarDisponibilidadeUseCase) Executar(in ConsultarDisponibilidadeInput) ([]availability.TimeBlock, error) {
	p, err := uc.providerRepo.BuscarPorID(in.ProviderID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProviderNaoEncontrado
	}

	excecao, err := uc.excecaoRepo.BuscarPorData(in.ProviderID, in.Data)
	if err != nil {
		return nil, err
	}
	if excecao != nil {
		if excecao.Tipo == availability.TipoBloqueio {
			return nil, nil
		}
		return excecao.Blocos, nil
	}

	schedule, err := uc.scheduleRepo.Buscar(in.ProviderID)
	if err != nil {
		return nil, err
	}
	if schedule != nil {
		// A grade existe (o prestador já configurou ao menos uma vez): o dia
		// específico pode estar vazio de propósito — não cai no default.
		return schedule.BlocosDoDia(availability.DiaSemanaDe(in.Data)), nil
	}

	if !p.AceitaAgendamentos {
		return nil, nil
	}
	dia := availability.DiaSemanaDe(in.Data)
	if dia == availability.Domingo || dia == availability.Sabado {
		return nil, nil
	}
	return blocosComerciaisPadrao, nil
}
