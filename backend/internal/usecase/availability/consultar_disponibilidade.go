package availability

import (
	"time"

	"agendago/internal/domain/availability"
	"agendago/internal/domain/provider"
)

// blocosPadrao resolve o expediente default de uma data sem definição própria:
// o expediente configurado em HorariosPadrao, aplicado em dias úteis quando o
// prestador aceita agendamentos; vazio em fins de semana ou com a agenda
// desativada.
func blocosPadrao(p *provider.Provider, data time.Time) []availability.TimeBlock {
	if !p.AceitaAgendamentos {
		return nil
	}
	dia := availability.DiaSemanaDe(data)
	if dia == availability.Domingo || dia == availability.Sabado {
		return nil
	}
	return p.HorariosPadrao
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
	excecaoRepo  repositorioDateException
	providerRepo repositorioProvider
}

// NovoConsultarDisponibilidadeUseCase cria uma instância de ConsultarDisponibilidadeUseCase com os repositórios injetados.
func NovoConsultarDisponibilidadeUseCase(
	excecaoRepo repositorioDateException,
	providerRepo repositorioProvider,
) *ConsultarDisponibilidadeUseCase {
	return &ConsultarDisponibilidadeUseCase{excecaoRepo: excecaoRepo, providerRepo: providerRepo}
}

// Executar resolve os blocos efetivos de ProviderID em Data seguindo a ordem:
// definição própria da data → expediente padrão (dia comercial em dias úteis,
// se AceitaAgendamentos). Retorna slice vazia (nunca nil-erro) quando o
// prestador não atende no dia.
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

	return blocosPadrao(p, in.Data), nil
}
