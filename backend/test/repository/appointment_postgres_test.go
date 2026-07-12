//go:build integration

package repository_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/appointment"
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
)

func TestAppointmentPostgres(t *testing.T) {
	pool := novoPool(t)
	providerRepo := repository.NovoProviderPostgres(pool)
	clientRepo := repository.NovoClientPostgres(pool)
	repo := repository.NovoAppointmentPostgres(pool)

	providerID := "dddddddd-0000-0000-0000-000000000001"
	clientID := "dddddddd-0000-0000-0000-000000000002"
	p, _ := provider.Novo(providerID, "Prestador Agendamento", "prestador-ag@email.com", "hash")
	if err := providerRepo.Salvar(p); err != nil {
		t.Fatalf("salvar prestador: %v", err)
	}
	c, _ := client.NovoComConta(clientID, "Cliente Agendamento", "cliente-ag@email.com", "hash")
	if err := clientRepo.Salvar(c); err != nil {
		t.Fatalf("salvar cliente: %v", err)
	}

	agora := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	data := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)

	novo := func(id string, inicio, fim int) *appointment.Appointment {
		a, err := appointment.Novo(id, providerID, clientID, data, inicio, fim, agora, 24*time.Hour)
		if err != nil {
			t.Fatalf("agendamento inválido no teste: %v", err)
		}
		return a
	}

	t.Run("salva quando livre e reflete na busca", func(t *testing.T) {
		a := novo("eeeeeeee-0000-0000-0000-000000000001", 8*60, 9*60)
		if err := repo.SalvarSeLivre(a, agora); err != nil {
			t.Fatalf("esperava salvar, got: %v", err)
		}

		encontrado, err := repo.BuscarPorID(a.ID)
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if encontrado == nil || encontrado.Status != appointment.StatusSolicitado || encontrado.InicioMinutos != 8*60 {
			t.Errorf("esperava solicitação persistida, got: %+v", encontrado)
		}
	})

	t.Run("intervalo em conflito é barrado (anti-overbooking)", func(t *testing.T) {
		b := novo("eeeeeeee-0000-0000-0000-000000000002", 8*60+30, 9*60+30)
		if err := repo.SalvarSeLivre(b, agora); err != appointment.ErrConflitoHorario {
			t.Errorf("esperava ErrConflitoHorario, got: %v", err)
		}
	})

	t.Run("solicitação expirada não bloqueia o intervalo", func(t *testing.T) {
		depoisDoTTL := agora.Add(48 * time.Hour)
		b := novo("eeeeeeee-0000-0000-0000-000000000003", 8*60, 9*60)
		b.ExpiraEm = depoisDoTTL.Add(24 * time.Hour)
		if err := repo.SalvarSeLivre(b, depoisDoTTL); err != nil {
			t.Errorf("esperava salvar sobre solicitação expirada, got: %v", err)
		}
	})

	t.Run("atualizar persiste a transição de status", func(t *testing.T) {
		a, _ := repo.BuscarPorID("eeeeeeee-0000-0000-0000-000000000003")
		if err := a.Confirmar(agora.Add(49 * time.Hour)); err != nil {
			t.Fatalf("confirmar no domínio: %v", err)
		}
		if err := repo.Atualizar(a); err != nil {
			t.Fatalf("esperava atualizar, got: %v", err)
		}
		relido, _ := repo.BuscarPorID(a.ID)
		if relido.Status != appointment.StatusConfirmado {
			t.Errorf("esperava CONFIRMADO persistido, got: %s", relido.Status)
		}
	})

	t.Run("listagens por prestador, cliente e ocupantes do período", func(t *testing.T) {
		doPrestador, err := repo.ListarPorPrestador(providerID)
		if err != nil || len(doPrestador) != 2 {
			t.Errorf("esperava 2 agendamentos do prestador, got: %d (%v)", len(doPrestador), err)
		}

		doCliente, err := repo.ListarPorCliente(clientID)
		if err != nil || len(doCliente) != 2 {
			t.Errorf("esperava 2 agendamentos do cliente, got: %d (%v)", len(doCliente), err)
		}

		// só o CONFIRMADO ocupa: a primeira solicitação já venceu o TTL
		ocupantes, err := repo.ListarOcupantesPorPeriodo(providerID, data, data, agora.Add(72*time.Hour))
		if err != nil || len(ocupantes) != 1 {
			t.Errorf("esperava 1 ocupante, got: %d (%v)", len(ocupantes), err)
		}
	})
}
