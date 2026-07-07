package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/availability"
	ucavailability "agendago/internal/usecase/availability"
)

func TestListarExcecoes(t *testing.T) {
	t.Run("lista vazia inicialmente", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoListarExcecoesUseCase(repo)

		out, err := uc.Executar(ucavailability.ListarExcecoesInput{ProviderID: "provider-1"})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Excecoes) != 0 {
			t.Errorf("esperava lista vazia, got: %d", len(out.Excecoes))
		}
	})

	t.Run("lista populada após criar exceções", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		criar := ucavailability.NovoCriarExcecaoUseCase(repo)
		criar.Executar(ucavailability.CriarExcecaoInput{
			ProviderID: "provider-1", Data: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), Tipo: availability.TipoBloqueio,
		})
		criar.Executar(ucavailability.CriarExcecaoInput{
			ProviderID: "provider-1", Data: time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), Tipo: availability.TipoBloqueio,
		})

		uc := ucavailability.NovoListarExcecoesUseCase(repo)
		out, err := uc.Executar(ucavailability.ListarExcecoesInput{ProviderID: "provider-1"})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Excecoes) != 2 {
			t.Errorf("esperava 2 exceções, got: %d", len(out.Excecoes))
		}
	})
}
