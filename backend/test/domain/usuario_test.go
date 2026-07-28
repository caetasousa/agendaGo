package domain_test

import (
	"testing"

	"agendago/internal/domain/usuario"
)

func TestNovoUsuario(t *testing.T) {
	t.Run("cria usuário ativo com dados válidos", func(t *testing.T) {
		u, err := usuario.Novo("1", "joao@email.com", "11999998888", "hash")
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if u.Email != "joao@email.com" {
			t.Errorf("esperava email 'joao@email.com', got: %s", u.Email)
		}
		if !u.Ativo {
			t.Error("usuário deve nascer ativo")
		}
	})

	t.Run("retorna erro quando email é vazio", func(t *testing.T) {
		_, err := usuario.Novo("1", "", "11999998888", "hash")
		if err != usuario.ErrEmailObrigatorio {
			t.Errorf("esperava ErrEmailObrigatorio, got: %v", err)
		}
	})

	t.Run("retorna erro quando o hash de senha é vazio", func(t *testing.T) {
		_, err := usuario.Novo("1", "joao@email.com", "11999998888", "")
		if err != usuario.ErrSenhaObrigatoria {
			t.Errorf("esperava ErrSenhaObrigatoria, got: %v", err)
		}
	})

	t.Run("retorna erro quando o telefone tem menos de 8 dígitos", func(t *testing.T) {
		_, err := usuario.Novo("1", "joao@email.com", "1199", "hash")
		if err != usuario.ErrTelefoneObrigatorio {
			t.Errorf("esperava ErrTelefoneObrigatorio, got: %v", err)
		}
	})

	t.Run("aceita telefone formatado, contando só os dígitos", func(t *testing.T) {
		u, err := usuario.Novo("1", "joao@email.com", "(11) 99999-8888", "hash")
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if u.Telefone != "(11) 99999-8888" {
			t.Errorf("esperava o telefone preservado como veio, got: %s", u.Telefone)
		}
	})
}

func TestDefinirTelefoneUsuario(t *testing.T) {
	t.Run("atualiza o telefone quando válido", func(t *testing.T) {
		u, _ := usuario.Novo("1", "joao@email.com", "11999998888", "hash")
		if err := u.DefinirTelefone("11888887777"); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if u.Telefone != "11888887777" {
			t.Errorf("esperava telefone '11888887777', got: %s", u.Telefone)
		}
	})

	t.Run("recusa telefone inválido e mantém o anterior", func(t *testing.T) {
		u, _ := usuario.Novo("1", "joao@email.com", "11999998888", "hash")
		if err := u.DefinirTelefone("123"); err != usuario.ErrTelefoneObrigatorio {
			t.Errorf("esperava ErrTelefoneObrigatorio, got: %v", err)
		}
		if u.Telefone != "11999998888" {
			t.Errorf("esperava o telefone anterior preservado, got: %s", u.Telefone)
		}
	})
}

func TestDefinirSenhaUsuario(t *testing.T) {
	t.Run("troca o hash da senha", func(t *testing.T) {
		u, _ := usuario.Novo("1", "joao@email.com", "11999998888", "hash-antigo")
		if err := u.DefinirSenha("hash-novo"); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if u.SenhaHash != "hash-novo" {
			t.Errorf("esperava 'hash-novo', got: %s", u.SenhaHash)
		}
	})

	t.Run("recusa hash vazio e mantém o anterior", func(t *testing.T) {
		u, _ := usuario.Novo("1", "joao@email.com", "11999998888", "hash-antigo")
		if err := u.DefinirSenha(""); err != usuario.ErrSenhaObrigatoria {
			t.Errorf("esperava ErrSenhaObrigatoria, got: %v", err)
		}
		if u.SenhaHash != "hash-antigo" {
			t.Errorf("esperava o hash anterior preservado, got: %s", u.SenhaHash)
		}
	})
}

func TestBanirEReativarUsuario(t *testing.T) {
	t.Run("banir desativa o usuário", func(t *testing.T) {
		u, _ := usuario.Novo("1", "joao@email.com", "11999998888", "hash")
		u.Banir()
		if u.Ativo {
			t.Error("esperava o usuário inativo depois de banido")
		}
	})

	t.Run("reativar devolve o acesso", func(t *testing.T) {
		u, _ := usuario.Novo("1", "joao@email.com", "11999998888", "hash")
		u.Banir()
		u.Reativar()
		if !u.Ativo {
			t.Error("esperava o usuário ativo depois de reativado")
		}
	})
}
