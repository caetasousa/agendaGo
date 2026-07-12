package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/domain/appointment"
	ucappointment "agendago/internal/usecase/appointment"
)

// confirma cria e confirma um agendamento no ambiente, devolvendo o slot
// solicitado às 08:00 da segundaFutura.
func confirmarAgendamento(t *testing.T, amb *ambienteAgendamento) *appointment.Appointment {
	t.Helper()
	out, err := amb.solicitar.Executar(ucappointment.SolicitarInput{
		ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60, Agora: agoraDoTeste,
	})
	if err != nil {
		t.Fatalf("solicitação de base falhou: %v", err)
	}
	if err := amb.transicionar.Confirmar(ucappointment.TransicionarInput{
		AgendamentoID: out.ID, UsuarioID: "provider-1", Tipo: "provider", Agora: agoraDoTeste,
	}); err != nil {
		t.Fatalf("confirmação de base falhou: %v", err)
	}
	a, _ := amb.appointments.BuscarPorID(out.ID)
	// isola os testes de lembrete dos emails de solicitação/confirmação já disparados
	amb.mailer.Limpar()
	return a
}

func TestLembrar(t *testing.T) {
	t.Run("envia lembrete para agendamento confirmado que começa dentro de 24h", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		a := confirmarAgendamento(t, amb)

		// início às 08:00 de segundaFutura; 10h antes ainda está dentro da janela de 24h
		agora := a.InicioEm(time.UTC).Add(-10 * time.Hour)

		if err := amb.lembrar.Executar(agora); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		enviadas := amb.mailer.Enviadas()
		if len(enviadas) != 1 {
			t.Fatalf("esperava 1 lembrete enviado, got: %d", len(enviadas))
		}
		if enviadas[0].Para != "maria@email.com" {
			t.Errorf("esperava lembrete para maria@email.com, got: %s", enviadas[0].Para)
		}
	})

	t.Run("não envia fora da janela de antecedência", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		a := confirmarAgendamento(t, amb)

		// início em 48h: fora da janela de 24h
		agora := a.InicioEm(time.UTC).Add(-48 * time.Hour)

		if err := amb.lembrar.Executar(agora); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(amb.mailer.Enviadas()) != 0 {
			t.Errorf("esperava zero lembretes fora da janela, got: %d", len(amb.mailer.Enviadas()))
		}
	})

	t.Run("não envia depois que o horário já passou", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		a := confirmarAgendamento(t, amb)

		agora := a.InicioEm(time.UTC).Add(time.Hour)

		if err := amb.lembrar.Executar(agora); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(amb.mailer.Enviadas()) != 0 {
			t.Errorf("esperava zero lembretes após o início, got: %d", len(amb.mailer.Enviadas()))
		}
	})

	t.Run("não envia duas vezes para o mesmo agendamento", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		a := confirmarAgendamento(t, amb)
		agora := a.InicioEm(time.UTC).Add(-10 * time.Hour)

		amb.lembrar.Executar(agora)
		amb.lembrar.Executar(agora.Add(time.Minute))

		if len(amb.mailer.Enviadas()) != 1 {
			t.Errorf("esperava exatamente 1 lembrete no total, got: %d", len(amb.mailer.Enviadas()))
		}
	})

	t.Run("não envia para agendamento apenas solicitado (não confirmado)", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		out, err := amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60, Agora: agoraDoTeste,
		})
		if err != nil {
			t.Fatalf("solicitação de base falhou: %v", err)
		}
		a, _ := amb.appointments.BuscarPorID(out.ID)
		amb.mailer.Limpar()

		agora := a.InicioEm(time.UTC).Add(-10 * time.Hour)
		if err := amb.lembrar.Executar(agora); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(amb.mailer.Enviadas()) != 0 {
			t.Errorf("esperava zero lembretes para agendamento apenas solicitado, got: %d", len(amb.mailer.Enviadas()))
		}
	})
}
