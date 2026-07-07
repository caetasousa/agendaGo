package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/availability"
	ucavailability "agendago/internal/usecase/availability"
)

func TestRemoverExcecao(t *testing.T) {
	data := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)

	t.Run("remove exceção com sucesso", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		criar := ucavailability.NovoCriarExcecaoUseCase(repo)
		criada, _ := criar.Executar(ucavailability.CriarExcecaoInput{ProviderID: "provider-1", Data: data, Tipo: availability.TipoBloqueio})

		uc := ucavailability.NovoRemoverExcecaoUseCase(repo)
		err := uc.Executar(ucavailability.RemoverExcecaoInput{ProviderID: "provider-1", ExcecaoID: criada.ID})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		encontrada, _ := repo.BuscarPorID(criada.ID)
		if encontrada != nil {
			t.Error("esperava exceção removida")
		}
	})

	t.Run("retorna erro quando exceção não existe", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoRemoverExcecaoUseCase(repo)

		err := uc.Executar(ucavailability.RemoverExcecaoInput{ProviderID: "provider-1", ExcecaoID: "id-fantasma"})
		if err != ucavailability.ErrExcecaoNaoEncontrada {
			t.Errorf("esperava ErrExcecaoNaoEncontrada, got: %v", err)
		}
	})

	t.Run("retorna erro ao tentar remover exceção de outro prestador", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		criar := ucavailability.NovoCriarExcecaoUseCase(repo)
		criada, _ := criar.Executar(ucavailability.CriarExcecaoInput{ProviderID: "provider-1", Data: data, Tipo: availability.TipoBloqueio})

		uc := ucavailability.NovoRemoverExcecaoUseCase(repo)
		err := uc.Executar(ucavailability.RemoverExcecaoInput{ProviderID: "provider-2", ExcecaoID: criada.ID})
		if err != ucavailability.ErrExcecaoNaoEncontrada {
			t.Errorf("esperava ErrExcecaoNaoEncontrada, got: %v", err)
		}

		// confirma que a exceção do provider-1 não foi removida
		encontrada, _ := repo.BuscarPorID(criada.ID)
		if encontrada == nil {
			t.Error("esperava que a exceção de outro prestador permanecesse intacta")
		}
	})
}
