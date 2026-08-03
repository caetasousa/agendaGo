// Package analytics resume a agenda de um prestador em números: o funil de
// status do período e quanto do expediente ofertado virou atendimento.
//
// Tudo aqui é leitura pura — nenhuma métrica escreve no banco, nem sequer
// altera o objeto em memória. Vale inclusive para a expiração lazy: uma
// solicitação vencida CONTA como expirada no relatório, mas efetivar a
// transição continua sendo trabalho da listagem.
package analytics

import (
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/domain/availability"
	ucavailability "agendago/internal/usecase/availability"
)

// maxDiasPeriodo limita o período consultável de uma vez, como na agenda e nos
// slots. Sem ele, um `?de=1970-01-01` varreria a tabela inteira do prestador.
const maxDiasPeriodo = 92

// chaveData é o formato usado para casar agendamentos, compromissos e dias do
// expediente pelo mesmo dia.
const chaveData = "2006-01-02"

// MetricasInput identifica a agenda, o período (inclusivo) e o instante da
// consulta — Agora vem do chamador para a regra do TTL ser testável.
type MetricasInput struct {
	ProviderID string
	De         time.Time
	Ate        time.Time
	Agora      time.Time
}

// MetricasOutput é o retrato do período. O recorte é sempre pela DATA DO
// ATENDIMENTO, não pela data em que ele foi pedido: "julho" aqui quer dizer os
// horários de julho, mesmo os marcados em junho.
type MetricasOutput struct {
	De  time.Time
	Ate time.Time
	// PorStatus traz uma entrada para CADA status do ciclo de vida, inclusive
	// os zerados: um funil se lê tanto pelo que tem quanto pelo que não tem, e
	// omitir a chave obrigaria quem exibe a manter sua própria lista completa.
	PorStatus map[appointment.Status]int
	Total     int
	// MinutosOfertados é o expediente do período menos os compromissos
	// pessoais; MinutosReservados, quanto dele foi consumido por agendamento.
	MinutosOfertados  int
	MinutosReservados int
	// TaxaOcupacao e TaxaComparecimento são frações em [0,1], nil quando não há
	// base para calcular. Um ponteiro em vez de 0 porque "0%" e "não há o que
	// medir" são leituras diferentes: um prestador sem nenhum atendimento no
	// período não tem 0% de comparecimento, ele não tem comparecimento nenhum.
	TaxaOcupacao       *float64
	TaxaComparecimento *float64
}

// MetricasUseCase calcula o resumo analítico da agenda de um prestador:
// distribuição de status e ocupação do expediente no período.
type MetricasUseCase struct {
	agenda          resolvedorAgenda
	appointmentRepo repositorioAppointment
	ocupacaoRepo    repositorioOcupacao
}

// NovoMetricasUseCase cria uma instância de MetricasUseCase com as dependências injetadas.
func NovoMetricasUseCase(
	agenda resolvedorAgenda,
	appointmentRepo repositorioAppointment,
	ocupacaoRepo repositorioOcupacao,
) *MetricasUseCase {
	return &MetricasUseCase{agenda: agenda, appointmentRepo: appointmentRepo, ocupacaoRepo: ocupacaoRepo}
}

// Executar resume o período do prestador. Retorna ErrPeriodoInvalido para
// período invertido ou maior que maxDiasPeriodo, e propaga
// availability.ErrProviderNaoEncontrado quando a agenda não existe.
//
// O expediente do período é resolvido pela configuração ATUAL do prestador:
// exceções de data são históricas (guardadas por data), mas o expediente
// padrão e a duração não são versionados. Mudar o horário de trabalho hoje
// reescreve, portanto, o denominador de ontem — o preço de não manter um
// histórico de configuração, aceitável para uma métrica de tendência.
func (uc *MetricasUseCase) Executar(in MetricasInput) (*MetricasOutput, error) {
	if in.Ate.Before(in.De) || in.Ate.Sub(in.De) > maxDiasPeriodo*24*time.Hour {
		return nil, ErrPeriodoInvalido
	}

	agenda, err := uc.agenda.Executar(ucavailability.ConsultarAgendaInput{
		ProviderID: in.ProviderID,
		De:         in.De,
		Ate:        in.Ate,
		// É o dono medindo a própria agenda: fechá-la ao público não apaga o
		// expediente que ele de fato tinha.
		IncluirAgendaFechada: true,
	})
	if err != nil {
		return nil, err
	}

	agendamentos, err := uc.appointmentRepo.ListarPorPeriodo(in.ProviderID, in.De, in.Ate)
	if err != nil {
		return nil, err
	}
	compromissos, err := uc.ocupacaoRepo.ListarPorPeriodo(in.ProviderID, in.De, in.Ate)
	if err != nil {
		return nil, err
	}

	porStatus := contagemZerada()
	reservadosPorDia := make(map[string][]intervalo)
	for _, a := range agendamentos {
		porStatus[statusEfetivo(a, in.Agora)]++
		if a.ConsumiuHorario() {
			chave := a.Data.Format(chaveData)
			reservadosPorDia[chave] = append(reservadosPorDia[chave], intervalo{a.InicioMinutos, a.FimMinutos})
		}
	}

	compromissosPorDia := make(map[string][]intervalo)
	for _, o := range compromissos {
		chave := o.Data.Format(chaveData)
		compromissosPorDia[chave] = append(compromissosPorDia[chave], intervalo{o.InicioMinutos, o.FimMinutos})
	}

	var ofertados, reservados int
	for _, dia := range agenda.Dias {
		chave := dia.Data.Format(chaveData)
		// O compromisso pessoal sai do denominador: aquele intervalo nunca foi
		// ofertado, e contá-lo como ociosidade puniria quem bloqueou a tarde
		// para ir ao médico.
		ofertados += minutosDosBlocos(dia.Blocos) - minutosSobrepostos(dia.Blocos, compromissosPorDia[chave])
		// O que caiu fora do expediente atual não entra: só assim a razão
		// significa "quanto do que eu ofereci foi usado" e nunca passa de 100%.
		reservados += minutosSobrepostos(dia.Blocos, reservadosPorDia[chave])
	}

	realizados := porStatus[appointment.StatusRealizado]
	ausencias := porStatus[appointment.StatusNaoCompareceu]

	return &MetricasOutput{
		De:                 in.De,
		Ate:                in.Ate,
		PorStatus:          porStatus,
		Total:              len(agendamentos),
		MinutosOfertados:   ofertados,
		MinutosReservados:  reservados,
		TaxaOcupacao:       razao(reservados, ofertados),
		TaxaComparecimento: razao(realizados, realizados+ausencias),
	}, nil
}

// statusEfetivo aplica a expiração lazy sem tocar no agendamento: uma
// solicitação com TTL vencido conta como EXPIRADO, exatamente como a listagem
// mostraria. A cópia existe para reusar a regra do domínio sem herdar o efeito
// colateral dela — relatório não muda estado de nada.
func statusEfetivo(a *appointment.Appointment, agora time.Time) appointment.Status {
	copia := *a
	copia.ExpirarSeVencido(agora)
	return copia.Status
}

func contagemZerada() map[appointment.Status]int {
	contagem := make(map[appointment.Status]int, len(appointment.TodosOsStatus))
	for _, s := range appointment.TodosOsStatus {
		contagem[s] = 0
	}
	return contagem
}

// razao devolve parte/total, ou nil quando não há total — dividir por zero e
// responder 0% seria inventar um dado que ninguém mediu.
func razao(parte, total int) *float64 {
	if total <= 0 {
		return nil
	}
	r := float64(parte) / float64(total)
	return &r
}

// intervalo é um trecho de um dia, em minutos desde a meia-noite.
type intervalo struct {
	inicio int
	fim    int
}

func minutosDosBlocos(blocos []availability.TimeBlock) int {
	var total int
	for _, b := range blocos {
		total += b.FimMinutos - b.InicioMinutos
	}
	return total
}

// minutosSobrepostos soma quanto dos intervalos cai dentro dos blocos do dia.
//
// Somar as interseções par a par só não conta o mesmo minuto duas vezes porque
// nenhum dos dois lados se sobrepõe internamente: os blocos saem normalizados
// de availability, e agendamentos e compromissos não colidem entre si — é o
// que o anti-overbooking garante na escrita.
func minutosSobrepostos(blocos []availability.TimeBlock, intervalos []intervalo) int {
	var total int
	for _, b := range blocos {
		for _, i := range intervalos {
			inicio := max(b.InicioMinutos, i.inicio)
			fim := min(b.FimMinutos, i.fim)
			if fim > inicio {
				total += fim - inicio
			}
		}
	}
	return total
}
