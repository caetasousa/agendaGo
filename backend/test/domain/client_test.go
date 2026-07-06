package domain_test

import (
	"testing"

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
	t.Run("cria client convidado sem conta", func(t *testing.T) {
		c, err := client.NovoConvidado("1", "Maria Silva", "maria@email.com")
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if c.TemConta() {
			t.Error("esperava que o client convidado não tivesse conta")
		}
		if c.SenhaHash != "" {
			t.Error("esperava SenhaHash vazio para convidado")
		}
	})

	t.Run("retorna erro quando nome é vazio", func(t *testing.T) {
		_, err := client.NovoConvidado("1", "", "maria@email.com")
		if err != client.ErrNomeObrigatorio {
			t.Errorf("esperava ErrNomeObrigatorio, got: %v", err)
		}
	})
}
