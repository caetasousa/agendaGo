//go:build integration

package repository_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/appointment"
	"agendago/internal/domain/cancellation"
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
)

func TestCancellationPostgres(t *testing.T) {
	pool := novoPool(t)
	providerRepo := repository.NovoProviderPostgres(pool)
	clientRepo := repository.NovoClientPostgres(pool)
	appointmentRepo := repository.NovoAppointmentPostgres(pool)
	repo := repository.NovoCancellationPostgres(pool)

	providerID := "cccccccc-0000-0000-0000-000000000001"
	clientID := "cccccccc-0000-0000-0000-000000000002"
	p, _ := provider.Novo(providerID, "Prestador Cancelamento", "prestador-cancel@email.com", "11999998888", "hash")
	if err := providerRepo.Salvar(p); err != nil {
		t.Fatalf("salvar prestador: %v", err)
	}
	c, _ := client.NovoComConta(clientID, "Cliente Cancelamento", "cliente-cancel@email.com", "hash")
	if err := clientRepo.Salvar(c); err != nil {
		t.Fatalf("salvar cliente: %v", err)
	}

	agora := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	data := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)

	// novoAppointmentID cria um agendamento real e devolve o ID — a FK de
	// cancelamento_tokens.appointment_id exige que ele exista. inicioMinutos
	// varia por chamada para não colidir com o anti-overbooking (mesmo
	// prestador, mesma data).
	proximoInicio := 8 * 60
	novoAppointmentID := func(id string) string {
		inicio := proximoInicio
		proximoInicio += 60
		a, err := appointment.Novo(id, providerID, clientID, data, inicio, inicio+60, agora, 24*time.Hour)
		if err != nil {
			t.Fatalf("agendamento inválido no teste: %v", err)
		}
		if err := appointmentRepo.SalvarSeLivre(a, agora); err != nil {
			t.Fatalf("salvar agendamento: %v", err)
		}
		return id
	}

	t.Run("salva e busca token por hash, sem apagar na leitura", func(t *testing.T) {
		appointmentID := novoAppointmentID("dddddddd-0000-0000-0000-000000000001")
		tok := cancellation.Novo("hash-aaa", appointmentID, time.Hour)
		if err := repo.Salvar(tok); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		encontrado, err := repo.BuscarPorTokenHash("hash-aaa")
		if err != nil {
			t.Fatalf("esperava sucesso na busca, got: %v", err)
		}
		if encontrado == nil {
			t.Fatal("esperava encontrar o token salvo")
		}
		if encontrado.AppointmentID != appointmentID {
			t.Errorf("appointmentID inesperado: %s", encontrado.AppointmentID)
		}

		// a leitura não consome — o token continua disponível para o
		// cancelamento subsequente; Remover é quem invalida (uso único)
		denovo, err := repo.BuscarPorTokenHash("hash-aaa")
		if err != nil || denovo == nil {
			t.Errorf("esperava o token ainda presente na segunda leitura, got: %v (%v)", denovo, err)
		}
	})

	t.Run("retorna (nil, nil) quando o hash não existe", func(t *testing.T) {
		encontrado, err := repo.BuscarPorTokenHash("hash-inexistente")
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if encontrado != nil {
			t.Errorf("esperava nil, got: %v", encontrado)
		}
	})

	t.Run("Remover apaga o token, invalidando-o para reuso", func(t *testing.T) {
		appointmentID := novoAppointmentID("dddddddd-0000-0000-0000-000000000002")
		tok := cancellation.Novo("hash-remover", appointmentID, time.Hour)
		if err := repo.Salvar(tok); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		if err := repo.Remover("hash-remover"); err != nil {
			t.Fatalf("esperava sucesso ao remover, got: %v", err)
		}
		if ainda, _ := repo.BuscarPorTokenHash("hash-remover"); ainda != nil {
			t.Error("esperava o token removido")
		}
	})

	t.Run("RemoverExpirados apaga só os tokens vencidos", func(t *testing.T) {
		expiradoID := novoAppointmentID("dddddddd-0000-0000-0000-000000000003")
		validoID := novoAppointmentID("dddddddd-0000-0000-0000-000000000004")
		expirado := cancellation.Novo("hash-cancel-expirado", expiradoID, -time.Hour)
		valido := cancellation.Novo("hash-cancel-valido", validoID, time.Hour)
		if err := repo.Salvar(expirado); err != nil {
			t.Fatalf("esperava sucesso ao salvar expirado, got: %v", err)
		}
		if err := repo.Salvar(valido); err != nil {
			t.Fatalf("esperava sucesso ao salvar válido, got: %v", err)
		}

		if err := repo.RemoverExpirados(); err != nil {
			t.Fatalf("esperava sucesso na limpeza, got: %v", err)
		}

		if ainda, _ := repo.BuscarPorTokenHash("hash-cancel-expirado"); ainda != nil {
			t.Error("esperava o token expirado removido")
		}
		if sumiu, _ := repo.BuscarPorTokenHash("hash-cancel-valido"); sumiu == nil {
			t.Error("esperava o token válido preservado")
		}
	})
}
