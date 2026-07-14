//go:build integration

package repository_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/precadastro"
)

func TestPreCadastroPostgres(t *testing.T) {
	repo := repository.NovoPreCadastroPostgres(novoPool(t))

	t.Run("salva e consome um pré-cadastro, apagando-o", func(t *testing.T) {
		p := precadastro.Novo("hash-aaa", "Convidada Silva", "convidada@email.com", "11999998888", time.Hour)
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

	t.Run("BuscarPorTokenHash não apaga o registro", func(t *testing.T) {
		p := precadastro.Novo("hash-busca", "Convidada", "busca@email.com", "11999998888", time.Hour)
		if err := repo.Salvar(p); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		primeira, err := repo.BuscarPorTokenHash("hash-busca")
		if err != nil || primeira == nil {
			t.Fatalf("esperava encontrar na primeira busca, got: %v (%v)", primeira, err)
		}
		segunda, err := repo.BuscarPorTokenHash("hash-busca")
		if err != nil || segunda == nil {
			t.Errorf("esperava o token ainda presente na segunda busca, got: %v (%v)", segunda, err)
		}
	})

	t.Run("RemoverExpirados apaga só os tokens vencidos", func(t *testing.T) {
		expirado := precadastro.Novo("hash-expirado", "Vencida", "vencida@email.com", "11999998888", -time.Hour)
		valido := precadastro.Novo("hash-valido", "Valida", "valida@email.com", "11999998888", time.Hour)
		if err := repo.Salvar(expirado); err != nil {
			t.Fatalf("esperava sucesso ao salvar expirado, got: %v", err)
		}
		if err := repo.Salvar(valido); err != nil {
			t.Fatalf("esperava sucesso ao salvar válido, got: %v", err)
		}

		if err := repo.RemoverExpirados(); err != nil {
			t.Fatalf("esperava sucesso na limpeza, got: %v", err)
		}

		if ainda, _ := repo.BuscarPorTokenHash("hash-expirado"); ainda != nil {
			t.Error("esperava o token expirado removido")
		}
		if sumiu, _ := repo.BuscarPorTokenHash("hash-valido"); sumiu == nil {
			t.Error("esperava o token válido preservado")
		}
	})
}
