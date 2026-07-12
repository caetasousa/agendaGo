package domain_test

import (
	"testing"

	"agendago/internal/domain/admin"
	"agendago/internal/domain/client"
)

func TestNovoComConta(t *testing.T) {
	t.Run("cria client com conta e dados válidos", func(t *testing.T) {
		c, err := client.NovoComConta("1", "Maria Silva", "maria@email.com", "hash-da-senha")
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if c.Nome != "Maria Silva" {
			t.Errorf("esperava nome 'Maria Silva', got: %s", c.Nome)
		}
		if !c.TemConta() {
			t.Error("esperava que o client tivesse conta")
		}
	})

	t.Run("retorna erro quando nome é vazio", func(t *testing.T) {
		_, err := client.NovoComConta("1", "", "maria@email.com", "hash-da-senha")
		if err != client.ErrNomeObrigatorio {
			t.Errorf("esperava ErrNomeObrigatorio, got: %v", err)
		}
	})

	t.Run("retorna erro quando email é vazio", func(t *testing.T) {
		_, err := client.NovoComConta("1", "Maria Silva", "", "hash-da-senha")
		if err != client.ErrEmailObrigatorio {
			t.Errorf("esperava ErrEmailObrigatorio, got: %v", err)
		}
	})

	t.Run("retorna erro quando hash de senha é vazio", func(t *testing.T) {
		_, err := client.NovoComConta("1", "Maria Silva", "maria@email.com", "")
		if err != client.ErrSenhaObrigatoria {
			t.Errorf("esperava ErrSenhaObrigatoria, got: %v", err)
		}
	})
}

func TestNovoConvidado(t *testing.T) {
	t.Run("cria client convidado sem conta, com telefone de contato", func(t *testing.T) {
		c, err := client.NovoConvidado("1", "Maria Silva", "maria@email.com", "(11) 99999-8888")
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if c.TemConta() {
			t.Error("esperava que o client convidado não tivesse conta")
		}
		if c.SenhaHash != "" {
			t.Error("esperava SenhaHash vazio para convidado")
		}
		if c.Telefone != "(11) 99999-8888" {
			t.Errorf("esperava o telefone informado, got: %s", c.Telefone)
		}
	})

	t.Run("retorna erro quando nome é vazio", func(t *testing.T) {
		_, err := client.NovoConvidado("1", "", "maria@email.com", "11999998888")
		if err != client.ErrNomeObrigatorio {
			t.Errorf("esperava ErrNomeObrigatorio, got: %v", err)
		}
	})

	t.Run("exige telefone com validação leve (ao menos 8 dígitos)", func(t *testing.T) {
		if _, err := client.NovoConvidado("1", "Maria", "maria@email.com", ""); err != client.ErrTelefoneObrigatorio {
			t.Errorf("esperava ErrTelefoneObrigatorio para telefone vazio, got: %v", err)
		}
		if _, err := client.NovoConvidado("1", "Maria", "maria@email.com", "123"); err != client.ErrTelefoneObrigatorio {
			t.Errorf("esperava ErrTelefoneObrigatorio para telefone curto, got: %v", err)
		}
	})
}

func TestBanirReativarClient(t *testing.T) {
	c, _ := client.NovoComConta("1", "Maria", "maria@email.com", "hash")
	if !c.Ativo {
		t.Fatal("cliente deve nascer ativo")
	}
	c.Banir()
	if c.Ativo {
		t.Error("esperava cliente inativo após banir")
	}
	c.Reativar()
	if !c.Ativo {
		t.Error("esperava cliente ativo após reativar")
	}
}

func TestNovoAdmin(t *testing.T) {
	a, err := admin.Novo("1", "admin@agendago.dev", "hash")
	if err != nil {
		t.Fatalf("esperava sucesso, got: %v", err)
	}
	if a.Email != "admin@agendago.dev" {
		t.Errorf("esperava email preservado, got: %s", a.Email)
	}
	if _, err := admin.Novo("1", "", "hash"); err != admin.ErrEmailObrigatorio {
		t.Errorf("esperava ErrEmailObrigatorio, got: %v", err)
	}
	if _, err := admin.Novo("1", "admin@agendago.dev", ""); err != admin.ErrSenhaObrigatoria {
		t.Errorf("esperava ErrSenhaObrigatoria, got: %v", err)
	}
}
