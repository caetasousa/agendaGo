package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/email"
	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	"agendago/internal/domain/appointment"
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/session"
	ucappointment "agendago/internal/usecase/appointment"
	ucavailability "agendago/internal/usecase/availability"
	ucclient "agendago/internal/usecase/client"
)

// ambienteAgendamento monta o conjunto de usecases de agendamento sobre
// repositórios em memória, com um prestador ativo (expediente 08–12/14–18,
// atendimento de 60min, sem descanso) e um cliente cadastrados.
type ambienteAgendamento struct {
	consultarSlots       *ucappointment.ConsultarSlotsUseCase
	solicitar            *ucappointment.SolicitarUseCase
	solicitarConvidado   *ucappointment.SolicitarConvidadoUseCase
	transicionar         *ucappointment.TransicionarUseCase
	cancelarPorToken     *ucappointment.CancelarPorTokenUseCase
	consultarPreCadastro *ucclient.ConsultarPreCadastroUseCase
	concluirPreCadastro  *ucclient.ConcluirPreCadastroUseCase
	listar               *ucappointment.ListarUseCase
	lembrar              *ucappointment.LembrarUseCase
	appointments         *repository.AppointmentMemoria
	clients              *repository.ClientMemoria
	cancelamentos        *repository.CancellationMemoria
	prestador            *provider.Provider
	mailer               *email.MailerMemoria
}

func novoAmbienteAgendamento(t *testing.T) *ambienteAgendamento {
	t.Helper()

	providerRepo := repository.NovoProviderMemoria()
	p, _ := provider.Novo("provider-1", "João Silva", "joao@email.com", "11999998888", "hash")
	p.AtivarAgenda()
	providerRepo.Salvar(p)

	clientRepo := repository.NovoClientMemoria()
	c, _ := client.NovoComConta("client-1", "Maria Souza", "maria@email.com", "hash")
	clientRepo.Salvar(c)

	availabilityRepo := repository.NovoAvailabilityMemoria()
	appointments := repository.NovoAppointmentMemoria()
	cancelamentos := repository.NovoCancellationMemoria()

	mailer := email.NovaMailerMemoria()
	notificador := email.NovoNotificador(mailer, "http://localhost:5173", time.UTC, email.ExecutorSincrono)

	resolvedor := ucavailability.NovoConsultarDisponibilidadeUseCase(availabilityRepo, providerRepo)
	consultarSlots := ucappointment.NovoConsultarSlotsUseCase(resolvedor, appointments, providerRepo, time.UTC)
	solicitar := ucappointment.NovoSolicitarUseCase(consultarSlots, appointments, clientRepo, providerRepo, notificador, 24*time.Hour)
	preCadastros := repository.NovoPreCadastroMemoria()
	solicitarConvidado := ucappointment.NovoSolicitarConvidadoUseCase(solicitar, clientRepo, providerRepo, cancelamentos, preCadastros, notificador)
	transicionar := ucappointment.NovoTransicionarUseCase(appointments, providerRepo, clientRepo, cancelamentos, preCadastros, notificador, 24*time.Hour, time.UTC)
	cancelarPorToken := ucappointment.NovoCancelarPorTokenUseCase(appointments, cancelamentos, providerRepo, clientRepo, notificador, 24*time.Hour, time.UTC)
	hasher := security.NovoHasherArgon2id()
	consultarPreCadastro := ucclient.NovoConsultarPreCadastroUseCase(preCadastros)
	concluirPreCadastro := ucclient.NovoConcluirPreCadastroUseCase(clientRepo, providerRepo, preCadastros, hasher)
	listar := ucappointment.NovoListarUseCase(appointments, providerRepo, clientRepo)
	lembrar := ucappointment.NovoLembrarUseCase(appointments, providerRepo, clientRepo, notificador, time.UTC, 24*time.Hour)

	return &ambienteAgendamento{
		consultarSlots:       consultarSlots,
		solicitar:            solicitar,
		solicitarConvidado:   solicitarConvidado,
		transicionar:         transicionar,
		cancelarPorToken:     cancelarPorToken,
		consultarPreCadastro: consultarPreCadastro,
		concluirPreCadastro:  concluirPreCadastro,
		listar:               listar,
		lembrar:              lembrar,
		appointments:         appointments,
		clients:              clientRepo,
		cancelamentos:        cancelamentos,
		prestador:            p,
		mailer:               mailer,
	}
}

// Segunda-feira futura e um "agora" bem antes dela.
var (
	segundaFutura = time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	agoraDoTeste  = time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
)

func TestConsultarSlots(t *testing.T) {
	t.Run("dia útil no expediente padrão oferta slots de 60min", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		out, err := amb.consultarSlots.Executar(ucappointment.ConsultarSlotsInput{
			ProviderID: "provider-1", De: segundaFutura, Ate: segundaFutura, Agora: agoraDoTeste,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		// 08–12 e 14–18 com 60min sem buffer: 4 + 4 slots
		if len(out.Dias) != 1 || len(out.Dias[0].Slots) != 8 {
			t.Fatalf("esperava 8 slots, got: %+v", out.Dias)
		}
		if out.Dias[0].Slots[0].InicioMinutos != 8*60 {
			t.Errorf("esperava primeiro slot às 08:00, got: %d", out.Dias[0].Slots[0].InicioMinutos)
		}
	})

	t.Run("solicitação ocupando o horário remove o slot da oferta", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 9 * 60, Agora: agoraDoTeste,
		})

		out, _ := amb.consultarSlots.Executar(ucappointment.ConsultarSlotsInput{
			ProviderID: "provider-1", De: segundaFutura, Ate: segundaFutura, Agora: agoraDoTeste,
		})
		for _, s := range out.Dias[0].Slots {
			if s.InicioMinutos == 9*60 {
				t.Error("slot das 09:00 deveria ter saído da oferta")
			}
		}
		if len(out.Dias[0].Slots) != 7 {
			t.Errorf("esperava 7 slots restantes, got: %d", len(out.Dias[0].Slots))
		}
	})

	t.Run("dia passado e agenda desativada não ofertam slots", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		ontem := agoraDoTeste.AddDate(0, 0, -1)
		out, _ := amb.consultarSlots.Executar(ucappointment.ConsultarSlotsInput{
			ProviderID: "provider-1", De: ontem, Ate: ontem, Agora: agoraDoTeste,
		})
		if len(out.Dias[0].Slots) != 0 {
			t.Error("dia passado não deveria ofertar slots")
		}

		amb.prestador.DesativarAgenda()
		out, _ = amb.consultarSlots.Executar(ucappointment.ConsultarSlotsInput{
			ProviderID: "provider-1", De: segundaFutura, Ate: segundaFutura, Agora: agoraDoTeste,
		})
		if len(out.Dias[0].Slots) != 0 {
			t.Error("agenda desativada não deveria ofertar slots")
		}
	})

	t.Run("hoje só oferta horários que ainda não começaram", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		// segunda-feira 10:30 da manhã: 08, 09 e 10 já passaram
		agora := time.Date(2026, 8, 10, 10, 30, 0, 0, time.UTC)

		out, _ := amb.consultarSlots.Executar(ucappointment.ConsultarSlotsInput{
			ProviderID: "provider-1", De: segundaFutura, Ate: segundaFutura, Agora: agora,
		})
		if len(out.Dias[0].Slots) != 5 {
			t.Fatalf("esperava 5 slots futuros (11h e a tarde), got: %d", len(out.Dias[0].Slots))
		}
		if out.Dias[0].Slots[0].InicioMinutos != 11*60 {
			t.Errorf("esperava primeiro slot às 11:00, got: %d", out.Dias[0].Slots[0].InicioMinutos)
		}
	})
}

func TestSolicitarAgendamento(t *testing.T) {
	t.Run("cria a solicitação ocupando o slot", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		out, err := amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60, Agora: agoraDoTeste,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.Status != appointment.StatusSolicitado || out.FimMinutos != 9*60 {
			t.Errorf("esperava SOLICITADO 08:00–09:00, got: %+v", out)
		}
	})

	t.Run("horário fora da oferta é rejeitado", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		_, err := amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 12 * 60, Agora: agoraDoTeste,
		})
		if err != ucappointment.ErrHorarioIndisponivel {
			t.Errorf("esperava ErrHorarioIndisponivel, got: %v", err)
		}
	})

	t.Run("dois clientes disputando o mesmo slot: o segundo falha (anti-overbooking)", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		if _, err := amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60, Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("primeira solicitação deveria passar, got: %v", err)
		}

		_, err := amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60, Agora: agoraDoTeste,
		})
		if err != ucappointment.ErrHorarioIndisponivel {
			t.Errorf("esperava ErrHorarioIndisponivel na disputa, got: %v", err)
		}
	})

	t.Run("solicitação expirada libera o slot para nova reserva", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60, Agora: agoraDoTeste,
		})

		// dois dias depois o TTL de 24h venceu — o slot volta a ficar livre
		depoisDoTTL := agoraDoTeste.Add(48 * time.Hour)
		if _, err := amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60, Agora: depoisDoTTL,
		}); err != nil {
			t.Errorf("esperava reservar o slot liberado pela expiração, got: %v", err)
		}
	})

	t.Run("solicitação bem-sucedida notifica o prestador por email", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		if _, err := amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60, Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		enviadas := amb.mailer.Enviadas()
		if len(enviadas) != 1 {
			t.Fatalf("esperava 1 email ao prestador, got: %d", len(enviadas))
		}
		if enviadas[0].Para != "joao@email.com" {
			t.Errorf("esperava email para o prestador joao@email.com, got: %s", enviadas[0].Para)
		}
	})

	t.Run("slot indisponível não notifica ninguém", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		_, err := amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 12 * 60, Agora: agoraDoTeste,
		})
		if err != ucappointment.ErrHorarioIndisponivel {
			t.Fatalf("esperava ErrHorarioIndisponivel, got: %v", err)
		}
		if len(amb.mailer.Enviadas()) != 0 {
			t.Errorf("esperava zero emails para solicitação rejeitada, got: %d", len(amb.mailer.Enviadas()))
		}
	})
}

func TestTransicionarAgendamento(t *testing.T) {
	solicitarPadrao := func(t *testing.T, amb *ambienteAgendamento) string {
		t.Helper()
		out, err := amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60, Agora: agoraDoTeste,
		})
		if err != nil {
			t.Fatalf("solicitação de base falhou: %v", err)
		}
		return out.ID
	}

	t.Run("prestador confirma e depois marca como realizado", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		id := solicitarPadrao(t, amb)

		if err := amb.transicionar.Confirmar(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("esperava confirmar, got: %v", err)
		}

		depoisDoAtendimento := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
		if err := amb.transicionar.MarcarRealizado(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: depoisDoAtendimento,
		}); err != nil {
			t.Fatalf("esperava marcar realizado, got: %v", err)
		}
	})

	t.Run("outro prestador não enxerga o agendamento", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		id := solicitarPadrao(t, amb)

		err := amb.transicionar.Confirmar(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "provider-2", Tipo: session.TipoProvider, Agora: agoraDoTeste,
		})
		if err != ucappointment.ErrAgendamentoNaoEncontrado {
			t.Errorf("esperava ErrAgendamentoNaoEncontrado, got: %v", err)
		}
	})

	t.Run("confirmar após o TTL expira a solicitação", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		id := solicitarPadrao(t, amb)

		err := amb.transicionar.Confirmar(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: agoraDoTeste.Add(25 * time.Hour),
		})
		if err != appointment.ErrSolicitacaoExpirada {
			t.Errorf("esperava ErrSolicitacaoExpirada, got: %v", err)
		}
		a, _ := amb.appointments.BuscarPorID(id)
		if a.Status != appointment.StatusExpirado {
			t.Errorf("esperava EXPIRADO persistido, got: %s", a.Status)
		}
	})

	t.Run("cliente cancela a própria solicitação pendente", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		id := solicitarPadrao(t, amb)

		if err := amb.transicionar.Cancelar(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "client-1", Tipo: session.TipoClient, Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("esperava cancelar, got: %v", err)
		}
	})

	t.Run("prestador não cancela solicitação pendente — recusa", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		id := solicitarPadrao(t, amb)

		err := amb.transicionar.Cancelar(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: agoraDoTeste,
		})
		if err != appointment.ErrTransicaoInvalida {
			t.Errorf("esperava ErrTransicaoInvalida, got: %v", err)
		}

		if err := amb.transicionar.Recusar(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("esperava recusar, got: %v", err)
		}
	})

	t.Run("prestador marca não comparecimento após o horário", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		id := solicitarPadrao(t, amb)
		amb.transicionar.Confirmar(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: agoraDoTeste,
		})

		// antes do horário não dá para registrar ausência
		err := amb.transicionar.MarcarNaoCompareceu(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: agoraDoTeste,
		})
		if err != appointment.ErrAtendimentoNaoIniciado {
			t.Errorf("esperava ErrAtendimentoNaoIniciado antes do horário, got: %v", err)
		}

		depoisDoAtendimento := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
		if err := amb.transicionar.MarcarNaoCompareceu(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: depoisDoAtendimento,
		}); err != nil {
			t.Fatalf("esperava marcar não comparecimento, got: %v", err)
		}
		a, _ := amb.appointments.BuscarPorID(id)
		if a.Status != appointment.StatusNaoCompareceu {
			t.Errorf("esperava NAO_COMPARECEU persistido, got: %s", a.Status)
		}
	})

	t.Run("cancelar confirmado em cima da hora é bloqueado", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		id := solicitarPadrao(t, amb)
		amb.transicionar.Confirmar(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: agoraDoTeste,
		})

		umaHoraAntes := time.Date(2026, 8, 10, 7, 0, 0, 0, time.UTC)
		err := amb.transicionar.Cancelar(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "client-1", Tipo: session.TipoClient, Agora: umaHoraAntes,
		})
		if err != appointment.ErrAntecedenciaInsuficiente {
			t.Errorf("esperava ErrAntecedenciaInsuficiente, got: %v", err)
		}
	})

	t.Run("confirmar notifica o cliente por email", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		id := solicitarPadrao(t, amb)
		amb.mailer.Limpar() // descarta o email de solicitação já disparado

		if err := amb.transicionar.Confirmar(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("esperava confirmar, got: %v", err)
		}

		enviadas := amb.mailer.Enviadas()
		if len(enviadas) != 1 {
			t.Fatalf("esperava 1 email ao cliente, got: %d", len(enviadas))
		}
		if enviadas[0].Para != "maria@email.com" {
			t.Errorf("esperava email para o cliente maria@email.com, got: %s", enviadas[0].Para)
		}
	})

	t.Run("recusar notifica o cliente por email", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		id := solicitarPadrao(t, amb)
		amb.mailer.Limpar()

		if err := amb.transicionar.Recusar(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("esperava recusar, got: %v", err)
		}

		enviadas := amb.mailer.Enviadas()
		if len(enviadas) != 1 {
			t.Fatalf("esperava 1 email ao cliente, got: %d", len(enviadas))
		}
		if enviadas[0].Para != "maria@email.com" {
			t.Errorf("esperava email para o cliente maria@email.com, got: %s", enviadas[0].Para)
		}
	})

	t.Run("cancelamento pelo cliente notifica o prestador", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		id := solicitarPadrao(t, amb)
		amb.mailer.Limpar()

		if err := amb.transicionar.Cancelar(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "client-1", Tipo: session.TipoClient, Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("esperava cancelar, got: %v", err)
		}

		enviadas := amb.mailer.Enviadas()
		if len(enviadas) != 1 {
			t.Fatalf("esperava 1 email ao prestador, got: %d", len(enviadas))
		}
		if enviadas[0].Para != "joao@email.com" {
			t.Errorf("esperava email para o prestador joao@email.com, got: %s", enviadas[0].Para)
		}
	})

	t.Run("cancelamento pelo prestador notifica o cliente", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		id := solicitarPadrao(t, amb)
		amb.transicionar.Confirmar(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: agoraDoTeste,
		})
		amb.mailer.Limpar()

		antecedenciaOk := agoraDoTeste
		if err := amb.transicionar.Cancelar(ucappointment.TransicionarInput{
			AgendamentoID: id, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: antecedenciaOk,
		}); err != nil {
			t.Fatalf("esperava cancelar, got: %v", err)
		}

		enviadas := amb.mailer.Enviadas()
		if len(enviadas) != 1 {
			t.Fatalf("esperava 1 email ao cliente, got: %d", len(enviadas))
		}
		if enviadas[0].Para != "maria@email.com" {
			t.Errorf("esperava email para o cliente maria@email.com (cancelado pelo prestador), got: %s", enviadas[0].Para)
		}
	})
}

func TestListarAgendamentos(t *testing.T) {
	t.Run("lista com nomes das duas pontas e expira solicitações vencidas", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60, Agora: agoraDoTeste,
		})

		depoisDoTTL := agoraDoTeste.Add(48 * time.Hour)
		out, err := amb.listar.DoPrestador(ucappointment.ListarInput{UsuarioID: "provider-1", Agora: depoisDoTTL})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Agendamentos) != 1 {
			t.Fatalf("esperava 1 agendamento, got: %d", len(out.Agendamentos))
		}
		a := out.Agendamentos[0]
		if a.Status != appointment.StatusExpirado {
			t.Errorf("esperava EXPIRADO após o TTL, got: %s", a.Status)
		}
		if a.NomeCliente != "Maria Souza" || a.NomePrestador != "João Silva" {
			t.Errorf("esperava nomes preenchidos, got: %+v", a)
		}

		doCliente, _ := amb.listar.DoCliente(ucappointment.ListarInput{UsuarioID: "client-1", Agora: depoisDoTTL})
		if len(doCliente.Agendamentos) != 1 {
			t.Errorf("esperava 1 agendamento na visão do cliente, got: %d", len(doCliente.Agendamentos))
		}
	})
}

func TestSolicitarConvidado(t *testing.T) {
	dados := func(inicio int) ucappointment.SolicitarConvidadoInput {
		return ucappointment.SolicitarConvidadoInput{
			ProviderID:    "provider-1",
			Data:          segundaFutura,
			InicioMinutos: inicio,
			Nome:          "Convidada Silva",
			Email:         "convidada@email.com",
			Telefone:      "(11) 99999-8888",
			Agora:         agoraDoTeste,
		}
	}

	t.Run("cria um convidado novo, guarda o contato e reserva o slot", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		out, err := amb.solicitarConvidado.Executar(dados(8 * 60))
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.Status != appointment.StatusSolicitado || out.FimMinutos != 9*60 {
			t.Errorf("esperava SOLICITADO 08:00–09:00, got: %+v", out)
		}

		criado, _ := amb.clients.BuscarPorEmail("convidada@email.com")
		if criado == nil {
			t.Fatal("esperava o convidado persistido")
		}
		if criado.TemConta() {
			t.Error("convidado não deveria ter conta")
		}
		if criado.Telefone != "(11) 99999-8888" {
			t.Errorf("esperava o telefone de contato guardado, got: %s", criado.Telefone)
		}

		// o prestador enxerga nome, email e telefone do convidado
		doPrestador, _ := amb.listar.DoPrestador(ucappointment.ListarInput{UsuarioID: "provider-1", Agora: agoraDoTeste})
		a := doPrestador.Agendamentos[0]
		if a.NomeCliente != "Convidada Silva" || a.EmailCliente != "convidada@email.com" || a.TelefoneCliente != "(11) 99999-8888" {
			t.Errorf("esperava contato do convidado visível ao prestador, got: %+v", a)
		}
	})

	t.Run("segundo agendamento com o mesmo email reusa o cliente, não duplica", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		if _, err := amb.solicitarConvidado.Executar(dados(8 * 60)); err != nil {
			t.Fatalf("primeiro agendamento deveria passar, got: %v", err)
		}
		if _, err := amb.solicitarConvidado.Executar(dados(9 * 60)); err != nil {
			t.Fatalf("segundo agendamento deveria passar, got: %v", err)
		}

		// os dois agendamentos existem, mas para o mesmo convidado
		doPrestador, _ := amb.listar.DoPrestador(ucappointment.ListarInput{UsuarioID: "provider-1", Agora: agoraDoTeste})
		if len(doPrestador.Agendamentos) != 2 {
			t.Fatalf("esperava 2 agendamentos, got: %d", len(doPrestador.Agendamentos))
		}
		if doPrestador.Agendamentos[0].EmailCliente != "convidada@email.com" ||
			doPrestador.Agendamentos[1].EmailCliente != "convidada@email.com" {
			t.Error("os dois agendamentos deveriam apontar para o mesmo convidado")
		}
	})

	t.Run("telefone inválido é rejeitado antes de reservar", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		in := dados(8 * 60)
		in.Telefone = "123"
		_, err := amb.solicitarConvidado.Executar(in)
		if err != client.ErrTelefoneObrigatorio {
			t.Errorf("esperava ErrTelefoneObrigatorio, got: %v", err)
		}
		if criado, _ := amb.clients.BuscarPorEmail("convidada@email.com"); criado != nil {
			t.Error("não deveria persistir convidado com telefone inválido")
		}
	})

	t.Run("cliente banido não agenda como convidado", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		banido, _ := client.NovoConvidado("banido-1", "Banido", "banido@email.com", "11999998888")
		banido.Banir()
		amb.clients.Salvar(banido)

		in := dados(8 * 60)
		in.Email = "banido@email.com"
		_, err := amb.solicitarConvidado.Executar(in)
		if err != ucappointment.ErrClientInativo {
			t.Errorf("esperava ErrClientInativo, got: %v", err)
		}
	})

	t.Run("e-mail de conta registrada é rejeitado — sem posse do e-mail não se agenda na conta alheia", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		// "maria@email.com" é a conta registrada do ambiente
		in := dados(8 * 60)
		in.Email = "maria@email.com"
		_, err := amb.solicitarConvidado.Executar(in)
		if err != ucappointment.ErrEmailTemConta {
			t.Errorf("esperava ErrEmailTemConta, got: %v", err)
		}

		// nada foi reservado na conta da vítima
		doCliente, _ := amb.listar.DoCliente(ucappointment.ListarInput{UsuarioID: "client-1", Agora: agoraDoTeste})
		if len(doCliente.Agendamentos) != 0 {
			t.Errorf("não deveria criar agendamento na conta registrada, got: %d", len(doCliente.Agendamentos))
		}
	})
}

func TestSolicitarClienteBanido(t *testing.T) {
	t.Run("cliente banido com sessão ativa não agenda pela rota autenticada", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		banido, _ := amb.clients.BuscarPorID("client-1")
		banido.Banir()
		amb.clients.Atualizar(banido)

		_, err := amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60, Agora: agoraDoTeste,
		})
		if err != ucappointment.ErrClientInativo {
			t.Errorf("esperava ErrClientInativo, got: %v", err)
		}
	})
}
