package ocupacao

import (
	"time"

	domocupacao "agendago/internal/domain/ocupacao"

	"github.com/google/uuid"
)

// CriarInput descreve o compromisso a registrar. Agora vem do chamador para a
// regra ser testável.
type CriarInput struct {
	ProviderID    string
	Data          time.Time
	InicioMinutos int
	FimMinutos    int
	Titulo        string
	Agora         time.Time
}

// CriarUseCase registra um compromisso pessoal na agenda do prestador.
type CriarUseCase struct {
	ocupacoes    repositorioOcupacao
	agendamentos repositorioAppointment
}

// NovoCriarUseCase cria uma instância de CriarUseCase com os repositórios injetados.
func NovoCriarUseCase(ocupacoes repositorioOcupacao, agendamentos repositorioAppointment) *CriarUseCase {
	return &CriarUseCase{ocupacoes: ocupacoes, agendamentos: agendamentos}
}

// Executar valida o intervalo, recusa sobreposição com agendamento ativo e
// persiste o compromisso.
//
// Não há regra de antecedência aqui, diferente do agendamento: criar
// compromisso só REDUZ a oferta e não desmarca ninguém, então bloquear "para
// hoje daqui a uma hora" atrapalharia sem proteger nada.
//
// Retorna ErrConflitoComAgendamento quando o horário já tem cliente marcado —
// o prestador cancela o agendamento primeiro, porque desmarcar cliente por
// conta própria não é decisão do sistema.
func (uc *CriarUseCase) Executar(in CriarInput) (*domocupacao.Ocupacao, error) {
	o, err := domocupacao.Nova(
		uuid.NewString(), in.ProviderID, in.Data,
		in.InicioMinutos, in.FimMinutos, in.Titulo, domocupacao.OrigemManual,
	)
	if err != nil {
		return nil, err
	}

	// Só o próprio dia interessa: um agendamento de outra data nunca colide.
	ocupantes, err := uc.agendamentos.ListarOcupantesPorPeriodo(in.ProviderID, in.Data, in.Data, in.Agora)
	if err != nil {
		return nil, err
	}
	for _, a := range ocupantes {
		if o.Colide(a.InicioMinutos, a.FimMinutos) {
			return nil, domocupacao.ErrConflitoComAgendamento
		}
	}

	if err := uc.ocupacoes.Salvar(o); err != nil {
		return nil, err
	}
	return o, nil
}
