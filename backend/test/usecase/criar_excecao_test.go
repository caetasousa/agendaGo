package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/availability"
	ucavailability "agendago/internal/usecase/availability"
)

func TestCriarExcecao(t *testing.T) {
	data := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)

	t.Run("cria exceção de bloqueio", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoCriarExcecaoUseCase(repo)

		out, err := uc.Executar(ucavailability.CriarExcecaoInput{
			ProviderID: "provider-1", Data: data, Tipo: availability.TipoBloqueio,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.ID == "" {
			t.Error("esperava ID gerado")
		}
	})

	t.Run("cria exceção extra com blocos", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoCriarExcecaoUseCase(repo)

		out, err := uc.Executar(ucavailability.CriarExcecaoInput{
			ProviderID: "provider-1", Data: data, Tipo: availability.TipoExtra,
			Blocos: []ucavailability.BlocoInput{{InicioMinutos: 600, FimMinutos: 660}},
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Blocos) != 1 {
			t.Error("esperava 1 bloco")
		}
	})

	t.Run("retorna erro quando já existe exceção para a data", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoCriarExcecaoUseCase(repo)

		uc.Executar(ucavailability.CriarExcecaoInput{ProviderID: "provider-1", Data: data, Tipo: availability.TipoBloqueio})
		_, err := uc.Executar(ucavailability.CriarExcecaoInput{ProviderID: "provider-1", Data: data, Tipo: availability.TipoBloqueio})
		if err != ucavailability.ErrExcecaoJaExiste {
			t.Errorf("esperava ErrExcecaoJaExiste, got: %v", err)
		}
	})

	t.Run("retorna erro quando extra não tem blocos", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoCriarExcecaoUseCase(repo)

		_, err := uc.Executar(ucavailability.CriarExcecaoInput{ProviderID: "provider-1", Data: data, Tipo: availability.TipoExtra})
		if err != availability.ErrExtraSemBlocos {
			t.Errorf("esperava ErrExtraSemBlocos, got: %v", err)
		}
	})
}
