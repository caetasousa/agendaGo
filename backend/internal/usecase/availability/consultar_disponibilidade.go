package availability

import (
	"time"

	"agendago/internal/domain/availability"
	"agendago/internal/domain/provider"
)

// blocosPadrao resolve o expediente default de uma data sem definição própria:
// o expediente configurado em HorariosPadrao, aplicado em dias úteis quando o
// prestador está ativo; vazio em fins de semana ou com o prestador banido pelo
// admin. A agenda desativada (AceitaAgendamentos=false) só zera o expediente
// para o público — o dono continua enxergando os próprios horários quando
// incluirAgendaFechada é true.
func blocosPadrao(p *provider.Provider, data time.Time, incluirAgendaFechada bool) []availability.TimeBlock {
	if !p.Ativo || (!p.AceitaAgendamentos && !incluirAgendaFechada) {
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
	// IncluirAgendaFechada resolve o expediente mesmo com AceitaAgendamentos
	// desligado — só para o próprio prestador operando a sua agenda. Exceções
	// de data (bloqueio explícito) valem também para o dono.
	IncluirAgendaFechada bool
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

	return blocosPadrao(p, in.Data, in.IncluirAgendaFechada), nil
}
