package usecase_test

import (
	"errors"
	"testing"
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/domain/ocupacao"
	ucanalytics "agendago/internal/usecase/analytics"
	ucavailability "agendago/internal/usecase/availability"
	"agendago/test/repository/memoria"
)

// A semana de referência dos testes: segunda 2026-08-10 a domingo 2026-08-16.
// O expediente padrão de um prestador novo (08–12 e 14–18) dá 480 minutos por
// dia útil, ou 2400 na semana — os cinco dias úteis do período.
var (
	segundaMetricas = time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	domingoMetricas = time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	// agoraMetricas é depois do período inteiro: os atendimentos já aconteceram.
	agoraMetricas = time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)
)

type ambienteMetricas struct {
	uc           *ucanalytics.MetricasUseCase
	agendamentos *memoria.AppointmentMemoria
	ocupacoes    *memoria.OcupacaoMemoria
}

func novoAmbienteMetricas(t *testing.T) *ambienteMetricas {
	t.Helper()
	usuarios, membros, providers := fakesDePrestador()
	criarPrestador(usuarios, membros, providers, "provider-1", "João Silva", "joao@email.com", "11999998888", senhaDeTeste)

	agendamentos := memoria.NovoAppointmentMemoria()
	ocupacoes := memoria.NovoOcupacaoMemoria()
	consultarAgenda := ucavailability.NovoConsultarAgendaUseCase(memoria.NovoAvailabilityMemoria(), providers, membros)

	return &ambienteMetricas{
		uc:           ucanalytics.NovoMetricasUseCase(consultarAgenda, agendamentos, ocupacoes),
		agendamentos: agendamentos,
		ocupacoes:    ocupacoes,
	}
}

// agendar persiste um agendamento no dia e horário pedidos, já no status
// desejado. O TTL é largo para a solicitação não vencer sozinha — quem testa
// expiração passa um curto de propósito.
func (a *ambienteMetricas) agendar(t *testing.T, id string, dia time.Time, inicio, fim int, status appointment.Status, ttl time.Duration) {
	t.Helper()
	criacao := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)

	ag, err := appointment.Novo(id, "provider-1", "client-1", dia, inicio, fim, criacao, ttl)
	if err != nil {
		t.Fatalf("agendamento de teste inválido: %v", err)
	}

	switch status {
	case appointment.StatusSolicitado:
	case appointment.StatusCancelado:
		// desistência do pedido pendente: não exige antecedência
		if err := ag.Cancelar(agoraMetricas, 24*time.Hour, time.UTC); err != nil {
			t.Fatalf("cancelar: %v", err)
		}
	case appointment.StatusRecusado:
		if err := ag.Recusar(criacao); err != nil {
			t.Fatalf("recusar: %v", err)
		}
	default:
		if err := ag.Confirmar(criacao); err != nil {
			t.Fatalf("confirmar: %v", err)
		}
		if status == appointment.StatusRealizado {
			if err := ag.MarcarRealizado(agoraMetricas, time.UTC); err != nil {
				t.Fatalf("marcar realizado: %v", err)
			}
		}
		if status == appointment.StatusNaoCompareceu {
			if err := ag.MarcarNaoCompareceu(agoraMetricas, time.UTC); err != nil {
				t.Fatalf("marcar ausência: %v", err)
			}
		}
	}

	if err := a.agendamentos.SalvarSeLivre(ag, criacao); err != nil {
		t.Fatalf("salvar agendamento: %v", err)
	}
}

func (a *ambienteMetricas) executar(t *testing.T) *ucanalytics.MetricasOutput {
	t.Helper()
	out, err := a.uc.Executar(ucanalytics.MetricasInput{
		ProviderID: "provider-1",
		De:         segundaMetricas,
		Ate:        domingoMetricas,
		Agora:      agoraMetricas,
	})
	if err != nil {
		t.Fatalf("esperava sucesso, got: %v", err)
	}
	return out
}

func taxa(t *testing.T, valor *float64) float64 {
	t.Helper()
	if valor == nil {
		t.Fatal("esperava taxa calculada, got: nil")
	}
	return *valor
}

func TestMetricasFunil(t *testing.T) {
	t.Run("conta cada status do período e traz os zerados", func(t *testing.T) {
		amb := novoAmbienteMetricas(t)
		amb.agendar(t, "a-1", segundaMetricas, 540, 600, appointment.StatusRealizado, 30*24*time.Hour)
		amb.agendar(t, "a-2", segundaMetricas.AddDate(0, 0, 1), 540, 600, appointment.StatusNaoCompareceu, 30*24*time.Hour)
		amb.agendar(t, "a-3", segundaMetricas.AddDate(0, 0, 2), 540, 600, appointment.StatusCancelado, 30*24*time.Hour)
		amb.agendar(t, "a-4", segundaMetricas.AddDate(0, 0, 3), 540, 600, appointment.StatusConfirmado, 30*24*time.Hour)

		out := amb.executar(t)

		if out.Total != 4 {
			t.Errorf("esperava 4 agendamentos no total, got: %d", out.Total)
		}
		if len(out.PorStatus) != len(appointment.TodosOsStatus) {
			t.Errorf("esperava uma entrada por status (%d), got: %d", len(appointment.TodosOsStatus), len(out.PorStatus))
		}
		esperado := map[appointment.Status]int{
			appointment.StatusRealizado:     1,
			appointment.StatusNaoCompareceu: 1,
			appointment.StatusCancelado:     1,
			appointment.StatusConfirmado:    1,
			appointment.StatusSolicitado:    0,
			appointment.StatusRecusado:      0,
			appointment.StatusExpirado:      0,
		}
		for status, quantidade := range esperado {
			if out.PorStatus[status] != quantidade {
				t.Errorf("esperava %d em %s, got: %d", quantidade, status, out.PorStatus[status])
			}
		}
	})

	t.Run("solicitação vencida conta como expirada sem persistir a transição", func(t *testing.T) {
		amb := novoAmbienteMetricas(t)
		amb.agendar(t, "a-1", segundaMetricas, 540, 600, appointment.StatusSolicitado, time.Hour)

		out := amb.executar(t)

		if out.PorStatus[appointment.StatusExpirado] != 1 || out.PorStatus[appointment.StatusSolicitado] != 0 {
			t.Errorf("esperava a solicitação vencida contada como EXPIRADO, got: %+v", out.PorStatus)
		}
		guardado, _ := amb.agendamentos.BuscarPorID("a-1")
		if guardado.Status != appointment.StatusSolicitado {
			t.Errorf("relatório não deve escrever: esperava SOLICITADO guardado, got: %s", guardado.Status)
		}
	})

	t.Run("agendamento fora do período não entra", func(t *testing.T) {
		amb := novoAmbienteMetricas(t)
		amb.agendar(t, "a-1", segundaMetricas.AddDate(0, 0, -1), 540, 600, appointment.StatusRealizado, 30*24*time.Hour)

		if out := amb.executar(t); out.Total != 0 {
			t.Errorf("esperava período vazio, got: %d", out.Total)
		}
	})
}

func TestMetricasOcupacao(t *testing.T) {
	t.Run("razão entre o reservado e o expediente da semana", func(t *testing.T) {
		amb := novoAmbienteMetricas(t)
		// 60 minutos em cada um dos três status que consomem horário.
		amb.agendar(t, "a-1", segundaMetricas, 540, 600, appointment.StatusRealizado, 30*24*time.Hour)
		amb.agendar(t, "a-2", segundaMetricas.AddDate(0, 0, 1), 540, 600, appointment.StatusNaoCompareceu, 30*24*time.Hour)
		amb.agendar(t, "a-3", segundaMetricas.AddDate(0, 0, 2), 540, 600, appointment.StatusConfirmado, 30*24*time.Hour)
		// Estes dois liberaram o horário: não podem contar como ocupação.
		amb.agendar(t, "a-4", segundaMetricas.AddDate(0, 0, 3), 540, 600, appointment.StatusCancelado, 30*24*time.Hour)
		amb.agendar(t, "a-5", segundaMetricas.AddDate(0, 0, 4), 540, 600, appointment.StatusSolicitado, time.Hour)

		out := amb.executar(t)

		if out.MinutosOfertados != 2400 {
			t.Errorf("esperava 2400 minutos ofertados (5 dias úteis × 480), got: %d", out.MinutosOfertados)
		}
		if out.MinutosReservados != 180 {
			t.Errorf("esperava 180 minutos reservados, got: %d", out.MinutosReservados)
		}
		if esperada := 180.0 / 2400.0; taxa(t, out.TaxaOcupacao) != esperada {
			t.Errorf("esperava ocupação de %.4f, got: %.4f", esperada, *out.TaxaOcupacao)
		}
	})

	t.Run("compromisso pessoal sai do expediente ofertado", func(t *testing.T) {
		amb := novoAmbienteMetricas(t)
		// Tarde inteira de segunda (14–18) bloqueada: 240 minutos a menos.
		compromisso, err := ocupacao.Nova("o-1", "provider-1", segundaMetricas, 840, 1080, "médico", ocupacao.OrigemManual)
		if err != nil {
			t.Fatalf("compromisso de teste inválido: %v", err)
		}
		amb.ocupacoes.Salvar(compromisso)

		if out := amb.executar(t); out.MinutosOfertados != 2160 {
			t.Errorf("esperava 2160 minutos ofertados (2400 − 240), got: %d", out.MinutosOfertados)
		}
	})

	t.Run("horário fora do expediente não infla a ocupação", func(t *testing.T) {
		amb := novoAmbienteMetricas(t)
		// Sábado não tem expediente padrão, e as 13h de uma segunda caem no
		// intervalo do almoço: nenhum dos dois foi ofertado.
		amb.agendar(t, "a-1", segundaMetricas.AddDate(0, 0, 5), 540, 600, appointment.StatusConfirmado, 30*24*time.Hour)
		amb.agendar(t, "a-2", segundaMetricas, 780, 840, appointment.StatusConfirmado, 30*24*time.Hour)

		out := amb.executar(t)

		if out.MinutosReservados != 0 {
			t.Errorf("esperava 0 minuto reservado dentro do expediente, got: %d", out.MinutosReservados)
		}
		if out.PorStatus[appointment.StatusConfirmado] != 2 {
			t.Errorf("o funil continua contando os dois, got: %d", out.PorStatus[appointment.StatusConfirmado])
		}
	})

	t.Run("agenda fechada ao público mantém o expediente do dono", func(t *testing.T) {
		// criarPrestador não ativa a agenda — é justamente o caso: fechar a
		// agenda por uma semana não pode zerar o denominador do relatório.
		amb := novoAmbienteMetricas(t)

		if out := amb.executar(t); out.MinutosOfertados != 2400 {
			t.Errorf("esperava o expediente do dono mesmo com agenda fechada, got: %d", out.MinutosOfertados)
		}
	})
}

func TestMetricasTaxas(t *testing.T) {
	t.Run("comparecimento é a fatia realizada dos atendimentos concluídos", func(t *testing.T) {
		amb := novoAmbienteMetricas(t)
		amb.agendar(t, "a-1", segundaMetricas, 540, 600, appointment.StatusRealizado, 30*24*time.Hour)
		amb.agendar(t, "a-2", segundaMetricas, 600, 660, appointment.StatusRealizado, 30*24*time.Hour)
		amb.agendar(t, "a-3", segundaMetricas, 660, 720, appointment.StatusNaoCompareceu, 30*24*time.Hour)

		if out := amb.executar(t); taxa(t, out.TaxaComparecimento) != 2.0/3.0 {
			t.Errorf("esperava 2/3 de comparecimento, got: %.4f", *out.TaxaComparecimento)
		}
	})

	t.Run("sem base para medir, a taxa vem nula em vez de zero", func(t *testing.T) {
		amb := novoAmbienteMetricas(t)
		amb.agendar(t, "a-1", segundaMetricas, 540, 600, appointment.StatusConfirmado, 30*24*time.Hour)

		out := amb.executar(t)

		if out.TaxaComparecimento != nil {
			t.Errorf("nenhum atendimento concluído: esperava nil, got: %.4f", *out.TaxaComparecimento)
		}
		if out.TaxaOcupacao == nil {
			t.Error("havia expediente ofertado: esperava ocupação calculada")
		}
	})
}

func TestMetricasPeriodoInvalido(t *testing.T) {
	amb := novoAmbienteMetricas(t)

	casos := map[string]struct{ de, ate time.Time }{
		"invertido": {de: domingoMetricas, ate: segundaMetricas},
		"longo demais": {
			de:  segundaMetricas,
			ate: segundaMetricas.AddDate(0, 0, 93),
		},
	}
	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			_, err := amb.uc.Executar(ucanalytics.MetricasInput{
				ProviderID: "provider-1",
				De:         caso.de,
				Ate:        caso.ate,
				Agora:      agoraMetricas,
			})
			if !errors.Is(err, ucanalytics.ErrPeriodoInvalido) {
				t.Errorf("esperava ErrPeriodoInvalido, got: %v", err)
			}
		})
	}
}
