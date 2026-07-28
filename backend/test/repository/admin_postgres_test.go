//go:build integration

package repository_test

import (
	"testing"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/admin"
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/usuario"
)

func TestAdminPostgres(t *testing.T) {
	pool := novoPool(t)
	repo := repository.NovoAdminPostgres(pool)

	t.Run("salva e busca por email e id", func(t *testing.T) {
		a, _ := admin.Novo("aaaaaaaa-1111-1111-1111-111111111111", "admin@agendago.dev", "hash-1")
		if err := repo.Salvar(a); err != nil {
			t.Fatalf("esperava salvar, got: %v", err)
		}

		porEmail, _ := repo.BuscarPorEmail("admin@agendago.dev")
		if porEmail == nil || porEmail.ID != a.ID {
			t.Errorf("esperava encontrar por email, got: %+v", porEmail)
		}
		porID, _ := repo.BuscarPorID(a.ID)
		if porID == nil || porID.Email != "admin@agendago.dev" {
			t.Errorf("esperava encontrar por id, got: %+v", porID)
		}
	})

	t.Run("upsert por email mantém o id e atualiza o hash", func(t *testing.T) {
		a1, _ := admin.Novo("aaaaaaaa-1111-1111-1111-111111111111", "admin@agendago.dev", "hash-1")
		repo.Salvar(a1)
		a2, _ := admin.Novo("bbbbbbbb-2222-2222-2222-222222222222", "admin@agendago.dev", "hash-2")
		if err := repo.Salvar(a2); err != nil {
			t.Fatalf("esperava upsert, got: %v", err)
		}

		encontrado, _ := repo.BuscarPorEmail("admin@agendago.dev")
		if encontrado.ID != a1.ID {
			t.Errorf("esperava manter o id original, got: %s", encontrado.ID)
		}
		if encontrado.SenhaHash != "hash-2" {
			t.Errorf("esperava hash atualizado, got: %s", encontrado.SenhaHash)
		}
	})

	t.Run("busca de email inexistente devolve (nil, nil)", func(t *testing.T) {
		encontrado, err := repo.BuscarPorEmail("ninguem@agendago.dev")
		if err != nil || encontrado != nil {
			t.Errorf("esperava (nil, nil), got: %+v (%v)", encontrado, err)
		}
	})
}

func TestModeracaoPostgres(t *testing.T) {
	pool := novoPool(t)
	providerRepo := repository.NovoProviderPostgres(pool)
	usuarioRepo := repository.NovoUsuarioPostgres(pool)
	clientRepo := repository.NovoClientPostgres(pool)

	// O banimento é da CONTA, não da agenda: desde a V14 ele mora em usuarios.
	t.Run("prestador nasce ativo e o banimento persiste na conta", func(t *testing.T) {
		const id = "cccccccc-1111-1111-1111-111111111111"
		p, _ := provider.Novo(id, "Prestador Mod")
		providerRepo.Salvar(p)
		u, _ := usuario.Novo(id, "moderado@email.com", "11999998888", "hash-da-senha")
		usuarioRepo.Salvar(u)

		encontrado, _ := usuarioRepo.BuscarPorID(id)
		if !encontrado.Ativo {
			t.Fatal("conta do prestador deveria nascer ativa")
		}

		encontrado.Banir()
		if err := usuarioRepo.Atualizar(encontrado); err != nil {
			t.Fatalf("esperava atualizar, got: %v", err)
		}
		relido, _ := usuarioRepo.BuscarPorID(id)
		if relido.Ativo {
			t.Error("esperava conta inativa persistida")
		}
	})

	t.Run("Listar de clients traz só quem tem conta, com o status ativo", func(t *testing.T) {
		comConta, _ := client.NovoComConta("dddddddd-1111-1111-1111-111111111111", "Com Conta", "com-conta@email.com", "hash")
		clientRepo.Salvar(comConta)
		convidado, _ := client.NovoConvidado("dddddddd-2222-2222-2222-222222222222", "Convidado", "convidado@email.com", "11999998888")
		clientRepo.Salvar(convidado)

		lista, _, err := clientRepo.Listar(paginaPadrao)
		if err != nil {
			t.Fatalf("esperava listar, got: %v", err)
		}
		for _, c := range lista {
			if c.ID == convidado.ID {
				t.Error("convidado (sem conta) não deveria aparecer na moderação")
			}
		}

		comConta.Banir()
		clientRepo.Atualizar(comConta)
		relido, _ := clientRepo.BuscarPorID(comConta.ID)
		if relido.Ativo {
			t.Error("esperava cliente inativo persistido")
		}
	})
}
