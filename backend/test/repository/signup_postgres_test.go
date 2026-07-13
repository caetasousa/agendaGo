//go:build integration

package repository_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/signup"
)

func TestSignupPostgres(t *testing.T) {
	repo := repository.NovoSignupPostgres(novoPool(t))

	t.Run("salva e consome um cadastro pendente, apagando-o", func(t *testing.T) {
		p := signup.Novo("hash-aaa", "Maria", "maria@email.com", "11999998888", "senha-hash", time.Hour)
		if err := repo.Salvar(p); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		consumido, err := repo.Consumir("hash-aaa")
		if err != nil {
			t.Fatalf("esperava sucesso ao consumir, got: %v", err)
		}
		if consumido == nil || consumido.Email != "maria@email.com" || consumido.Telefone != "11999998888" {
			t.Fatalf("cadastro pendente inesperado: %+v", consumido)
		}

		// uso único: segundo consumo devolve nil
		if segundo, _ := repo.Consumir("hash-aaa"); segundo != nil {
			t.Error("esperava nil ao consumir um pendente já consumido")
		}
	})

	t.Run("retorna (nil, nil) quando o hash não existe", func(t *testing.T) {
		consumido, err := repo.Consumir("hash-inexistente")
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if consumido != nil {
			t.Errorf("esperava nil, got: %v", consumido)
		}
	})

	t.Run("remove pendentes anteriores do mesmo email", func(t *testing.T) {
		repo.Salvar(signup.Novo("hash-1", "Ana", "ana@email.com", "11999998888", "h", time.Hour))
		repo.Salvar(signup.Novo("hash-2", "Ana", "ana@email.com", "11999998888", "h", time.Hour))

		if err := repo.RemoverPorEmail("ana@email.com"); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if p, _ := repo.Consumir("hash-1"); p != nil {
			t.Error("esperava primeiro pendente removido")
		}
		if p, _ := repo.Consumir("hash-2"); p != nil {
			t.Error("esperava segundo pendente removido")
		}
	})

	t.Run("remove pendentes expirados", func(t *testing.T) {
		repo.Salvar(signup.Novo("hash-exp", "Exp", "exp@email.com", "11999998888", "h", -time.Hour))
		repo.Salvar(signup.Novo("hash-val", "Val", "val@email.com", "11999998888", "h", time.Hour))

		if err := repo.RemoverExpirados(); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if p, _ := repo.Consumir("hash-exp"); p != nil {
			t.Error("esperava pendente expirado removido")
		}
		if p, _ := repo.Consumir("hash-val"); p == nil {
			t.Error("esperava pendente válido mantido")
		}
	})
}
