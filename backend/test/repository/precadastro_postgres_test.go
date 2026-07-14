//go:build integration

package repository_test

import (
	"testing"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/precadastro"
)

func TestPreCadastroPostgres(t *testing.T) {
	repo := repository.NovoPreCadastroPostgres(novoPool(t))

	t.Run("salva e consome um pré-cadastro, apagando-o", func(t *testing.T) {
		p := precadastro.Novo("hash-aaa", "Convidada Silva", "convidada@email.com", "11999998888")
		if err := repo.Salvar(p); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		consumido, err := repo.Consumir("hash-aaa")
		if err != nil {
			t.Fatalf("esperava sucesso ao consumir, got: %v", err)
		}
		if consumido == nil || consumido.Nome != "Convidada Silva" || consumido.Email != "convidada@email.com" || consumido.Telefone != "11999998888" {
			t.Fatalf("pré-cadastro inesperado: %+v", consumido)
		}

		// uso único: segundo consumo devolve nil
		if segundo, _ := repo.Consumir("hash-aaa"); segundo != nil {
			t.Error("esperava nil ao consumir um pré-cadastro já consumido")
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
}
