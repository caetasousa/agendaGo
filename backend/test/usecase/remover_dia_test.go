package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/availability"
	ucavailability "agendago/internal/usecase/availability"
)

func TestRemoverDia(t *testing.T) {
	data := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)

	t.Run("remove a definição da data com sucesso", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		excecao, _ := availability.NovaDateException("exc-1", "provider-1", data, availability.TipoBloqueio, nil)
		repo.SalvarExcecao(excecao)

		uc := ucavailability.NovoRemoverDiaUseCase(repo)
		if err := uc.Executar(ucavailability.RemoverDiaInput{ProviderID: "provider-1", Data: data}); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		encontrada, _ := repo.BuscarPorData("provider-1", data)
		if encontrada != nil {
			t.Error("esperava definição removida")
		}
	})

	t.Run("retorna ErrDiaNaoDefinido quando a data não tem definição própria", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoRemoverDiaUseCase(repo)

		err := uc.Executar(ucavailability.RemoverDiaInput{ProviderID: "provider-1", Data: data})
		if err != ucavailability.ErrDiaNaoDefinido {
			t.Errorf("esperava ErrDiaNaoDefinido, got: %v", err)
		}
	})

	t.Run("não remove definição de outro prestador na mesma data", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		excecao, _ := availability.NovaDateException("exc-1", "provider-2", data, availability.TipoBloqueio, nil)
		repo.SalvarExcecao(excecao)

		uc := ucavailability.NovoRemoverDiaUseCase(repo)
		err := uc.Executar(ucavailability.RemoverDiaInput{ProviderID: "provider-1", Data: data})
		if err != ucavailability.ErrDiaNaoDefinido {
			t.Errorf("esperava ErrDiaNaoDefinido, got: %v", err)
		}

		encontrada, _ := repo.BuscarPorData("provider-2", data)
		if encontrada == nil {
			t.Error("esperava que a definição do outro prestador continuasse existindo")
		}
	})
}
