package domain_test

import (
	"testing"
	"time"

	"agendago/internal/domain/appointment"
)

func novoAgendamento(t *testing.T, agora time.Time) *appointment.Appointment {
	t.Helper()
	data := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	a, err := appointment.Novo("ag-1", "provider-1", "client-1", data, 9*60, 10*60, agora, 24*time.Hour)
	if err != nil {
		t.Fatalf("esperava sucesso ao criar, got: %v", err)
	}
	return a
}

func TestNovoAppointment(t *testing.T) {
	agora := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)

	t.Run("nasce SOLICITADO ocupando o intervalo, com expiração em agora+TTL", func(t *testing.T) {
		a := novoAgendamento(t, agora)
		if a.Status != appointment.StatusSolicitado {
			t.Errorf("esperava SOLICITADO, got: %s", a.Status)
		}
		if !a.Ocupa(agora) {
			t.Error("solicitação recém-criada deveria ocupar o intervalo")
		}
		if !a.ExpiraEm.Equal(agora.Add(24 * time.Hour)) {
			t.Errorf("esperava expiração em agora+24h, got: %v", a.ExpiraEm)
		}
	})

	t.Run("valida ids e intervalo", func(t *testing.T) {
		data := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
		if _, err := appointment.Novo("a", "", "c", data, 0, 60, agora, time.Hour); err != appointment.ErrProviderIDObrigatorio {
			t.Errorf("esperava ErrProviderIDObrigatorio, got: %v", err)
		}
		if _, err := appointment.Novo("a", "p", "", data, 0, 60, agora, time.Hour); err != appointment.ErrClientIDObrigatorio {
			t.Errorf("esperava ErrClientIDObrigatorio, got: %v", err)
		}
		if _, err := appointment.Novo("a", "p", "c", data, 600, 600, agora, time.Hour); err != appointment.ErrIntervaloInvalido {
			t.Errorf("esperava ErrIntervaloInvalido para duração zero, got: %v", err)
		}
		if _, err := appointment.Novo("a", "p", "c", data, 1400, 1500, agora, time.Hour); err != appointment.ErrIntervaloInvalido {
			t.Errorf("esperava ErrIntervaloInvalido cruzando a meia-noite, got: %v", err)
		}
	})
}

func TestCicloDeVida(t *testing.T) {
	agora := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)

	t.Run("confirmar solicitação pendente", func(t *testing.T) {
		a := novoAgendamento(t, agora)
		if err := a.Confirmar(agora.Add(time.Hour)); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if a.Status != appointment.StatusConfirmado {
			t.Errorf("esperava CONFIRMADO, got: %s", a.Status)
		}
	})

	t.Run("confirmar após o TTL retorna ErrSolicitacaoExpirada", func(t *testing.T) {
		a := novoAgendamento(t, agora)
		if err := a.Confirmar(agora.Add(25 * time.Hour)); err != appointment.ErrSolicitacaoExpirada {
			t.Errorf("esperava ErrSolicitacaoExpirada, got: %v", err)
		}
	})

	t.Run("solicitação expirada deixa de ocupar e vira EXPIRADO na leitura", func(t *testing.T) {
		a := novoAgendamento(t, agora)
		depois := agora.Add(25 * time.Hour)
		if a.Ocupa(depois) {
			t.Error("solicitação vencida não deveria ocupar o intervalo")
		}
		if !a.ExpirarSeVencido(depois) {
			t.Error("esperava a transição lazy para EXPIRADO")
		}
		if a.Status != appointment.StatusExpirado {
			t.Errorf("esperava EXPIRADO, got: %s", a.Status)
		}
	})

	t.Run("recusar solicitação pendente libera o intervalo", func(t *testing.T) {
		a := novoAgendamento(t, agora)
		if err := a.Recusar(agora); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if a.Status != appointment.StatusRecusado || a.Ocupa(agora) {
			t.Errorf("esperava RECUSADO sem ocupar, got: %s", a.Status)
		}
	})

	t.Run("cancelar confirmado com antecedência suficiente", func(t *testing.T) {
		a := novoAgendamento(t, agora)
		a.Confirmar(agora)
		// início: 2026-08-10 09:00 UTC; cancelando 9 dias antes
		if err := a.Cancelar(agora, 24*time.Hour, time.UTC); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if a.Status != appointment.StatusCancelado {
			t.Errorf("esperava CANCELADO, got: %s", a.Status)
		}
	})

	t.Run("cancelar confirmado dentro da janela mínima é bloqueado", func(t *testing.T) {
		a := novoAgendamento(t, agora)
		a.Confirmar(agora)
		emCimaDaHora := time.Date(2026, 8, 10, 8, 0, 0, 0, time.UTC) // 1h antes do início
		if err := a.Cancelar(emCimaDaHora, 24*time.Hour, time.UTC); err != appointment.ErrAntecedenciaInsuficiente {
			t.Errorf("esperava ErrAntecedenciaInsuficiente, got: %v", err)
		}
	})

	t.Run("cancelar solicitação pendente é livre, sem antecedência", func(t *testing.T) {
		a := novoAgendamento(t, agora)
		quaseNaHora := time.Date(2026, 8, 10, 8, 59, 0, 0, time.UTC)
		// mantém a solicitação viva até lá para o teste focar só na regra de antecedência
		a.ExpiraEm = quaseNaHora.Add(time.Hour)
		if err := a.Cancelar(quaseNaHora, 24*time.Hour, time.UTC); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
	})

	t.Run("realizado e não compareceu exigem confirmado com horário já iniciado", func(t *testing.T) {
		a := novoAgendamento(t, agora)
		a.Confirmar(agora)

		antesDoInicio := time.Date(2026, 8, 10, 8, 0, 0, 0, time.UTC)
		if err := a.MarcarRealizado(antesDoInicio, time.UTC); err != appointment.ErrAtendimentoNaoIniciado {
			t.Errorf("esperava ErrAtendimentoNaoIniciado, got: %v", err)
		}

		depoisDoInicio := time.Date(2026, 8, 10, 9, 30, 0, 0, time.UTC)
		if err := a.MarcarRealizado(depoisDoInicio, time.UTC); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if a.Status != appointment.StatusRealizado {
			t.Errorf("esperava REALIZADO, got: %s", a.Status)
		}

		if err := a.MarcarNaoCompareceu(depoisDoInicio, time.UTC); err != appointment.ErrTransicaoInvalida {
			t.Errorf("esperava ErrTransicaoInvalida sobre agendamento já concluído, got: %v", err)
		}
	})

	t.Run("transições inválidas são rejeitadas", func(t *testing.T) {
		a := novoAgendamento(t, agora)
		a.Recusar(agora)
		if err := a.Confirmar(agora); err != appointment.ErrTransicaoInvalida {
			t.Errorf("esperava ErrTransicaoInvalida, got: %v", err)
		}
		if err := a.Cancelar(agora, time.Hour, time.UTC); err != appointment.ErrTransicaoInvalida {
			t.Errorf("esperava ErrTransicaoInvalida, got: %v", err)
		}
	})
}
