package availability

import (
	"time"

	"agendago/internal/domain/availability"
)

// maxDiasAgenda limita o período consultável de uma vez — suficiente para o
// calendário navegar mês a mês sem permitir consultas desproporcionais.
const maxDiasAgenda = 92

// OrigemDia indica de onde veio a disponibilidade resolvida de uma data.
type OrigemDia string

const (
	// OrigemPadrao indica que a data segue o expediente padrão (sem definição própria).
	OrigemPadrao OrigemDia = "padrao"
	// OrigemBloqueio indica que a data foi marcada como indisponível pelo prestador.
	OrigemBloqueio OrigemDia = "bloqueio"
	// OrigemExtra indica que a data tem horários personalizados definidos pelo prestador.
	OrigemExtra OrigemDia = "extra"
)

// ConsultarAgendaInput define o prestador e o período (inclusivo) a resolver.
type ConsultarAgendaInput struct {
	ProviderID string
	De         time.Time
	Ate        time.Time
}

// DiaAgenda é a disponibilidade resolvida de uma data.
type DiaAgenda struct {
	Data   time.Time
	Origem OrigemDia
	Blocos []availability.TimeBlock
}

// ConsultarAgendaOutput contém a agenda resolvida do período, dia a dia.
type ConsultarAgendaOutput struct {
	AceitaAgendamentos bool
	Dias               []DiaAgenda
}

// ConsultarAgendaUseCase resolve a disponibilidade de um período inteiro para
// alimentar o calendário do prestador: cada dia sai como padrão, bloqueio ou
// horários personalizados, já com os blocos efetivos.
type ConsultarAgendaUseCase struct {
	excecaoRepo  repositorioDateException
	providerRepo repositorioProvider
}

// NovoConsultarAgendaUseCase cria uma instância de ConsultarAgendaUseCase com os repositórios injetados.
func NovoConsultarAgendaUseCase(
	excecaoRepo repositorioDateException,
	providerRepo repositorioProvider,
) *ConsultarAgendaUseCase {
	return &ConsultarAgendaUseCase{excecaoRepo: excecaoRepo, providerRepo: providerRepo}
}

// Executar resolve cada dia do período na ordem: definição própria da data →
// expediente padrão. Retorna ErrPeriodoInvalido para período invertido ou
// maior que maxDiasAgenda, e ErrProviderNaoEncontrado se o prestador não existe.
func (uc *ConsultarAgendaUseCase) Executar(in ConsultarAgendaInput) (*ConsultarAgendaOutput, error) {
	if in.Ate.Before(in.De) || in.Ate.Sub(in.De) > maxDiasAgenda*24*time.Hour {
		return nil, ErrPeriodoInvalido
	}

	p, err := uc.providerRepo.BuscarPorID(in.ProviderID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProviderNaoEncontrado
	}

	excecoes, err := uc.excecaoRepo.Listar(in.ProviderID)
	if err != nil {
		return nil, err
	}
	porData := make(map[string]*availability.DateException, len(excecoes))
	for _, e := range excecoes {
		porData[e.Data.Format("2006-01-02")] = e
	}

	var dias []DiaAgenda
	for d := in.De; !d.After(in.Ate); d = d.AddDate(0, 0, 1) {
		dia := DiaAgenda{Data: d, Origem: OrigemPadrao, Blocos: blocosPadrao(p, d)}
		if e, ok := porData[d.Format("2006-01-02")]; ok {
			if e.Tipo == availability.TipoBloqueio {
				dia.Origem = OrigemBloqueio
				dia.Blocos = nil
			} else {
				dia.Origem = OrigemExtra
				dia.Blocos = e.Blocos
			}
		}
		dias = append(dias, dia)
	}

	return &ConsultarAgendaOutput{AceitaAgendamentos: p.AceitaAgendamentos, Dias: dias}, nil
}
