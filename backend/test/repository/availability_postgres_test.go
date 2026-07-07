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
	p, _ := provider.Novo(id, "Prestador Disponibilidade", id+"@email.com", "hash-da-senha")
	if err := repo.Salvar(p); err != nil {
		t.Fatalf("esperava sucesso ao salvar provider, got: %v", err)
	}
}

func TestAvailabilityPostgresWeeklySchedule(t *testing.T) {
	pool := novoPool(t)
	providerRepo := repository.NovoProviderPostgres(pool)
	repo := repository.NovoAvailabilityPostgres(pool)

	t.Run("retorna (nil, nil) quando prestador nunca configurou a grade", func(t *testing.T) {
		novoProviderParaDisponibilidade(t, providerRepo, "aaaaaaaa-0000-0000-0000-000000000001")

		encontrada, err := repo.Buscar("aaaaaaaa-0000-0000-0000-000000000001")
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if encontrada != nil {
			t.Errorf("esperava nil, got: %v", encontrada)
		}
	})

	t.Run("salva a grade e reflete no Buscar", func(t *testing.T) {
		providerID := "aaaaaaaa-0000-0000-0000-000000000002"
		novoProviderParaDisponibilidade(t, providerRepo, providerID)

		bloco, _ := availability.NovoTimeBlock(8*60, 12*60)
		s, _ := availability.NovaWeeklySchedule(providerID, map[availability.DiaSemana][]availability.TimeBlock{
			availability.Segunda: {bloco},
		})
		if err := repo.Salvar(s); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		encontrada, err := repo.Buscar(providerID)
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if encontrada == nil {
			t.Fatal("esperava encontrar a grade salva")
		}
		blocos := encontrada.BlocosDoDia(availability.Segunda)
		if len(blocos) != 1 || blocos[0].InicioMinutos != 480 || blocos[0].FimMinutos != 720 {
			t.Errorf("esperava bloco 480-720 na segunda, got: %v", blocos)
		}
	})

	t.Run("salvar de novo substitui os blocos anteriores", func(t *testing.T) {
		providerID := "aaaaaaaa-0000-0000-0000-000000000003"
		novoProviderParaDisponibilidade(t, providerRepo, providerID)

		bloco1, _ := availability.NovoTimeBlock(8*60, 12*60)
		s1, _ := availability.NovaWeeklySchedule(providerID, map[availability.DiaSemana][]availability.TimeBlock{
			availability.Segunda: {bloco1},
		})
		repo.Salvar(s1)

		bloco2, _ := availability.NovoTimeBlock(9*60, 10*60)
		s2, _ := availability.NovaWeeklySchedule(providerID, map[availability.DiaSemana][]availability.TimeBlock{
			availability.Terca: {bloco2},
		})
		if err := repo.Salvar(s2); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		encontrada, _ := repo.Buscar(providerID)
		if len(encontrada.BlocosDoDia(availability.Segunda)) != 0 {
			t.Error("esperava que segunda-feira não tivesse mais blocos")
		}
		if len(encontrada.BlocosDoDia(availability.Terca)) != 1 {
			t.Error("esperava 1 bloco em terça-feira")
		}
	})
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
			t.Fatal("esperava encontrar a exceção")
		}
		if encontrada.Tipo != availability.TipoBloqueio {
			t.Errorf("esperava TipoBloqueio, got: %v", encontrada.Tipo)
		}
	})

	t.Run("salva extra com blocos e busca por ID", func(t *testing.T) {
		data := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
		bloco, _ := availability.NovoTimeBlock(10*60, 11*60)
		e, _ := availability.NovaDateException("cccccccc-0000-0000-0000-000000000002", providerID, data, availability.TipoExtra, []availability.TimeBlock{bloco})
		repo.SalvarExcecao(e)

		encontrada, err := repo.BuscarPorID("cccccccc-0000-0000-0000-000000000002")
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if encontrada == nil || len(encontrada.Blocos) != 1 {
			t.Fatalf("esperava exceção extra com 1 bloco, got: %v", encontrada)
		}
	})

	t.Run("retorna (nil, nil) quando não há exceção para a data", func(t *testing.T) {
		encontrada, err := repo.BuscarPorData(providerID, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if encontrada != nil {
			t.Errorf("esperava nil, got: %v", encontrada)
		}
	})

	t.Run("lista as exceções do prestador", func(t *testing.T) {
		excecoes, err := repo.Listar(providerID)
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if len(excecoes) != 2 {
			t.Errorf("esperava 2 exceções, got: %d", len(excecoes))
		}
	})

	t.Run("remove uma exceção", func(t *testing.T) {
		if err := repo.Remover("cccccccc-0000-0000-0000-000000000001"); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		encontrada, _ := repo.BuscarPorID("cccccccc-0000-0000-0000-000000000001")
		if encontrada != nil {
			t.Error("esperava exceção removida")
		}
	})

	t.Run("falha ao salvar data duplicada (constraint UNIQUE)", func(t *testing.T) {
		data := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		e1, _ := availability.NovaDateException("cccccccc-0000-0000-0000-000000000003", providerID, data, availability.TipoBloqueio, nil)
		e2, _ := availability.NovaDateException("cccccccc-0000-0000-0000-000000000004", providerID, data, availability.TipoBloqueio, nil)

		if err := repo.SalvarExcecao(e1); err != nil {
			t.Fatalf("esperava sucesso no primeiro salvar, got: %v", err)
		}
		if err := repo.SalvarExcecao(e2); err == nil {
			t.Error("esperava erro ao salvar data duplicada")
		}
	})
}
