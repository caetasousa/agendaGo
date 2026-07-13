//go:build integration

package repository_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/availability"
	"agendago/internal/domain/provider"
)

func novoProviderParaDisponibilidade(t *testing.T, repo *repository.ProviderPostgres, id string) {
	t.Helper()
	p, _ := provider.Novo(id, "Prestador Disponibilidade", id+"@email.com", "11999998888", "hash-da-senha")
	if err := repo.Salvar(p); err != nil {
		t.Fatalf("esperava sucesso ao salvar provider, got: %v", err)
	}
}

func TestAvailabilityPostgresDateException(t *testing.T) {
	pool := novoPool(t)
	providerRepo := repository.NovoProviderPostgres(pool)
	repo := repository.NovoAvailabilityPostgres(pool)

	providerID := "bbbbbbbb-0000-0000-0000-000000000001"
	novoProviderParaDisponibilidade(t, providerRepo, providerID)

	t.Run("salva bloqueio e busca por data", func(t *testing.T) {
		data := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
		e, _ := availability.NovaDateException("cccccccc-0000-0000-0000-000000000001", providerID, data, availability.TipoBloqueio, nil)
		if err := repo.SalvarExcecao(e); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		encontrada, err := repo.BuscarPorData(providerID, data)
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if encontrada == nil {
			t.Fatal("esperava encontrar a definição")
		}
		if encontrada.Tipo != availability.TipoBloqueio {
			t.Errorf("esperava TipoBloqueio, got: %v", encontrada.Tipo)
		}
	})

	t.Run("salva extra com blocos e devolve os blocos na busca", func(t *testing.T) {
		data := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
		bloco, _ := availability.NovoTimeBlock(10*60, 11*60)
		e, _ := availability.NovaDateException("cccccccc-0000-0000-0000-000000000002", providerID, data, availability.TipoExtra, []availability.TimeBlock{bloco})
		if err := repo.SalvarExcecao(e); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		encontrada, err := repo.BuscarPorData(providerID, data)
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if encontrada == nil || len(encontrada.Blocos) != 1 {
			t.Fatalf("esperava definição extra com 1 bloco, got: %v", encontrada)
		}
	})

	t.Run("retorna (nil, nil) quando a data não tem definição própria", func(t *testing.T) {
		encontrada, err := repo.BuscarPorData(providerID, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if encontrada != nil {
			t.Errorf("esperava nil, got: %v", encontrada)
		}
	})

	t.Run("lista as definições do prestador", func(t *testing.T) {
		excecoes, err := repo.Listar(providerID)
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if len(excecoes) != 2 {
			t.Errorf("esperava 2 definições, got: %d", len(excecoes))
		}
	})

	t.Run("salvar de novo na mesma data substitui tipo e blocos (upsert)", func(t *testing.T) {
		data := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		e1, _ := availability.NovaDateException("cccccccc-0000-0000-0000-000000000003", providerID, data, availability.TipoBloqueio, nil)
		if err := repo.SalvarExcecao(e1); err != nil {
			t.Fatalf("esperava sucesso no primeiro salvar, got: %v", err)
		}

		bloco, _ := availability.NovoTimeBlock(8*60, 12*60)
		e2, _ := availability.NovaDateException("cccccccc-0000-0000-0000-000000000004", providerID, data, availability.TipoExtra, []availability.TimeBlock{bloco})
		if err := repo.SalvarExcecao(e2); err != nil {
			t.Fatalf("esperava sucesso no upsert, got: %v", err)
		}

		encontrada, err := repo.BuscarPorData(providerID, data)
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if encontrada.Tipo != availability.TipoExtra || len(encontrada.Blocos) != 1 {
			t.Errorf("esperava definição substituída (extra com 1 bloco), got: %+v", encontrada)
		}
		// o id original da linha é mantido pelo upsert
		if encontrada.ID != "cccccccc-0000-0000-0000-000000000003" {
			t.Errorf("esperava id original mantido, got: %s", encontrada.ID)
		}
	})

	t.Run("remove uma definição", func(t *testing.T) {
		data := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
		if err := repo.Remover("cccccccc-0000-0000-0000-000000000001"); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		encontrada, _ := repo.BuscarPorData(providerID, data)
		if encontrada != nil {
			t.Error("esperava definição removida")
		}
	})
}
