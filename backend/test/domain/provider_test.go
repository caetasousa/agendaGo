package domain_test

import (
	"testing"

	"agendago/internal/domain/provider"
)

func TestNovo(t *testing.T) {
	t.Run("cria provider com dados válidos e agenda desativada por padrão", func(t *testing.T) {
		p, err := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if p.Nome != "João Silva" {
			t.Errorf("esperava nome 'João Silva', got: %s", p.Nome)
		}
		if p.AceitaAgendamentos {
			t.Error("agenda deve iniciar desativada")
		}
		if p.DescansoMinutos != 0 {
			t.Errorf("descanso deve iniciar em 0, got: %d", p.DescansoMinutos)
		}
	})

	t.Run("retorna erro quando nome é vazio", func(t *testing.T) {
		_, err := provider.Novo("1", "", "joao@email.com", "12345678")
		if err != provider.ErrNomeObrigatorio {
			t.Errorf("esperava ErrNomeObrigatorio, got: %v", err)
		}
	})

	t.Run("retorna erro quando email é vazio", func(t *testing.T) {
		_, err := provider.Novo("1", "João Silva", "", "12345678")
		if err != provider.ErrEmailObrigatorio {
			t.Errorf("esperava ErrEmailObrigatorio, got: %v", err)
		}
	})

	t.Run("retorna erro quando hash de senha é vazio", func(t *testing.T) {
		_, err := provider.Novo("1", "João Silva", "joao@email.com", "")
		if err != provider.ErrSenhaObrigatoria {
			t.Errorf("esperava ErrSenhaObrigatoria, got: %v", err)
		}
	})
}

func TestAgenda(t *testing.T) {
	t.Run("ativa a agenda do provider", func(t *testing.T) {
		p, _ := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
		p.AtivarAgenda()
		if !p.AceitaAgendamentos {
			t.Error("agenda deveria estar ativa")
		}
	})

	t.Run("desativa a agenda do provider", func(t *testing.T) {
		p, _ := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
		p.AtivarAgenda()
		p.DesativarAgenda()
		if p.AceitaAgendamentos {
			t.Error("agenda deveria estar desativada")
		}
	})
}

func TestDefinirDescanso(t *testing.T) {
	t.Run("define o tempo de descanso entre atendimentos", func(t *testing.T) {
		p, _ := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
		if err := p.DefinirDescanso(15); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if p.DescansoMinutos != 15 {
			t.Errorf("esperava 15 minutos, got: %d", p.DescansoMinutos)
		}
	})

	t.Run("retorna erro quando descanso é negativo", func(t *testing.T) {
		p, _ := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
		if err := p.DefinirDescanso(-1); err != provider.ErrDescansoInvalido {
			t.Errorf("esperava ErrDescansoInvalido, got: %v", err)
		}
	})
}
