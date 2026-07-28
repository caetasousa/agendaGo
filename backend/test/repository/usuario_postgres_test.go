//go:build integration

package repository_test

import (
	"testing"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/membro"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/usuario"
)

func TestUsuarioPostgres(t *testing.T) {
	pool := novoPool(t)
	repo := repository.NovoUsuarioPostgres(pool)

	t.Run("salva e busca usuário por email", func(t *testing.T) {
		u, _ := usuario.Novo("11111111-1111-1111-1111-111111111111", "joao@email.com", "11999998888", "hash")
		if err := repo.Salvar(u); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		encontrado, err := repo.BuscarPorEmail("joao@email.com")
		if err != nil {
			t.Fatalf("esperava sucesso na busca, got: %v", err)
		}
		if encontrado == nil {
			t.Fatal("esperava encontrar o usuário salvo")
		}
		if encontrado.ID != u.ID {
			t.Errorf("esperava ID %s, got: %s", u.ID, encontrado.ID)
		}
		if encontrado.SenhaHash != "hash" {
			t.Errorf("esperava o hash preservado, got: %s", encontrado.SenhaHash)
		}
		if !encontrado.Ativo {
			t.Error("esperava o usuário ativo")
		}
	})

	t.Run("busca por email inexistente devolve nil sem erro", func(t *testing.T) {
		encontrado, err := repo.BuscarPorEmail("ninguem@email.com")
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if encontrado != nil {
			t.Errorf("esperava nil, got: %+v", encontrado)
		}
	})

	t.Run("busca por id", func(t *testing.T) {
		u, _ := usuario.Novo("22222222-2222-2222-2222-222222222222", "maria@email.com", "11888887777", "hash")
		if err := repo.Salvar(u); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		encontrado, err := repo.BuscarPorID(u.ID)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if encontrado == nil || encontrado.Email != "maria@email.com" {
			t.Errorf("esperava o usuário maria@email.com, got: %+v", encontrado)
		}
	})

	t.Run("atualiza telefone e situação", func(t *testing.T) {
		u, _ := usuario.Novo("33333333-3333-3333-3333-333333333333", "carlos@email.com", "11777776666", "hash")
		if err := repo.Salvar(u); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		if err := u.DefinirTelefone("11555554444"); err != nil {
			t.Fatalf("esperava sucesso ao definir telefone, got: %v", err)
		}
		u.Banir()
		if err := repo.Atualizar(u); err != nil {
			t.Fatalf("esperava sucesso ao atualizar, got: %v", err)
		}

		encontrado, _ := repo.BuscarPorID(u.ID)
		if encontrado.Telefone != "11555554444" {
			t.Errorf("esperava telefone atualizado, got: %s", encontrado.Telefone)
		}
		if encontrado.Ativo {
			t.Error("esperava o usuário banido")
		}
	})

	t.Run("atualiza a senha sem tocar no resto", func(t *testing.T) {
		u, _ := usuario.Novo("44444444-4444-4444-4444-444444444444", "ana@email.com", "11666665555", "hash-antigo")
		if err := repo.Salvar(u); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		if err := repo.AtualizarSenha(u.ID, "hash-novo"); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		encontrado, _ := repo.BuscarPorID(u.ID)
		if encontrado.SenhaHash != "hash-novo" {
			t.Errorf("esperava 'hash-novo', got: %s", encontrado.SenhaHash)
		}
		if encontrado.Email != "ana@email.com" {
			t.Errorf("esperava o email intacto, got: %s", encontrado.Email)
		}
	})

	t.Run("email é único", func(t *testing.T) {
		primeiro, _ := usuario.Novo("55555555-5555-5555-5555-555555555555", "repetido@email.com", "11999998888", "hash")
		if err := repo.Salvar(primeiro); err != nil {
			t.Fatalf("esperava sucesso ao salvar o primeiro, got: %v", err)
		}

		segundo, _ := usuario.Novo("66666666-6666-6666-6666-666666666666", "repetido@email.com", "11999998888", "hash")
		if err := repo.Salvar(segundo); err == nil {
			t.Error("esperava erro de unicidade ao repetir o email")
		}
	})
}

func TestMembroPostgres(t *testing.T) {
	pool := novoPool(t)
	usuarios := repository.NovoUsuarioPostgres(pool)
	providers := repository.NovoProviderPostgres(pool)
	repo := repository.NovoMembroPostgres(pool)

	const (
		idUsuario  = "aaaaaaaa-0000-0000-0000-000000000001"
		idProvider = "bbbbbbbb-0000-0000-0000-000000000001"
	)

	// O vínculo tem FK para os dois lados: sem usuário e prestador de verdade
	// o INSERT nem chega a ser testado.
	u, _ := usuario.Novo(idUsuario, "dono@email.com", "11999998888", "hash")
	if err := usuarios.Salvar(u); err != nil {
		t.Fatalf("preparar usuário: %v", err)
	}
	p, _ := provider.Novo(idProvider, "Agenda do João")
	if err := providers.Salvar(p); err != nil {
		t.Fatalf("preparar prestador: %v", err)
	}

	t.Run("salva e busca o vínculo pelo usuário", func(t *testing.T) {
		m, err := membro.Novo("cccccccc-0000-0000-0000-000000000001", idUsuario, idProvider, membro.PapelDono)
		if err != nil {
			t.Fatalf("criar vínculo: %v", err)
		}
		if err := repo.Salvar(m); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		encontrado, err := repo.BuscarPorUsuario(idUsuario)
		if err != nil {
			t.Fatalf("esperava sucesso na busca, got: %v", err)
		}
		if encontrado == nil {
			t.Fatal("esperava encontrar o vínculo salvo")
		}
		if encontrado.ProviderID != idProvider {
			t.Errorf("esperava provider %s, got: %s", idProvider, encontrado.ProviderID)
		}
		if encontrado.Papel != membro.PapelDono {
			t.Errorf("esperava papel dono, got: %s", encontrado.Papel)
		}
	})

	t.Run("usuário sem vínculo devolve nil sem erro", func(t *testing.T) {
		encontrado, err := repo.BuscarPorUsuario("dddddddd-0000-0000-0000-000000000009")
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if encontrado != nil {
			t.Errorf("esperava nil, got: %+v", encontrado)
		}
	})

	t.Run("lista os vínculos de uma agenda", func(t *testing.T) {
		outro, _ := usuario.Novo("aaaaaaaa-0000-0000-0000-000000000002", "operador@email.com", "11888887777", "hash")
		if err := usuarios.Salvar(outro); err != nil {
			t.Fatalf("preparar usuário operador: %v", err)
		}
		m, _ := membro.Novo("cccccccc-0000-0000-0000-000000000002", outro.ID, idProvider, membro.PapelOperador)
		if err := repo.Salvar(m); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		membros, err := repo.ListarPorProvider(idProvider)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(membros) != 2 {
			t.Fatalf("esperava 2 vínculos na agenda, got: %d", len(membros))
		}
	})

	t.Run("o mesmo usuário não se vincula duas vezes à mesma agenda", func(t *testing.T) {
		m, _ := membro.Novo("cccccccc-0000-0000-0000-000000000003", idUsuario, idProvider, membro.PapelOperador)
		if err := repo.Salvar(m); err == nil {
			t.Error("esperava erro de unicidade no par (usuario, provider)")
		}
	})
}
