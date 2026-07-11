package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/availability"
	ucavailability "agendago/internal/usecase/availability"
)

func TestDefinirDia(t *testing.T) {
	data := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)

	t.Run("define bloqueio com sucesso", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoDefinirDiaUseCase(repo)

		out, err := uc.Executar(ucavailability.DefinirDiaInput{
			ProviderID: "provider-1", Data: data, Tipo: availability.TipoBloqueio,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.Tipo != availability.TipoBloqueio || len(out.Blocos) != 0 {
			t.Errorf("esperava bloqueio sem blocos, got: %+v", out)
		}
	})

	t.Run("define extra com blocos", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoDefinirDiaUseCase(repo)

		out, err := uc.Executar(ucavailability.DefinirDiaInput{
			ProviderID: "provider-1", Data: data, Tipo: availability.TipoExtra,
			Blocos: []ucavailability.BlocoInput{{InicioMinutos: 600, FimMinutos: 660}},
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Blocos) != 1 || out.Blocos[0].InicioMinutos != 600 {
			t.Errorf("esperava 1 bloco 600-660, got: %v", out.Blocos)
		}
	})

	t.Run("definir de novo substitui a definição anterior da data", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoDefinirDiaUseCase(repo)

		uc.Executar(ucavailability.DefinirDiaInput{
			ProviderID: "provider-1", Data: data, Tipo: availability.TipoBloqueio,
		})
		out, err := uc.Executar(ucavailability.DefinirDiaInput{
			ProviderID: "provider-1", Data: data, Tipo: availability.TipoExtra,
			Blocos: []ucavailability.BlocoInput{{InicioMinutos: 480, FimMinutos: 720}},
		})
		if err != nil {
			t.Fatalf("esperava sucesso no upsert, got: %v", err)
		}
		if out.Tipo != availability.TipoExtra {
			t.Errorf("esperava tipo extra após substituir, got: %v", out.Tipo)
		}

		excecoes, _ := repo.Listar("provider-1")
		if len(excecoes) != 1 {
			t.Fatalf("esperava 1 definição após upsert, got: %d", len(excecoes))
		}
		if excecoes[0].Tipo != availability.TipoExtra {
			t.Errorf("esperava a definição substituída (extra), got: %v", excecoes[0].Tipo)
		}
	})

	t.Run("retorna erro quando bloco é inválido", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoDefinirDiaUseCase(repo)

		_, err := uc.Executar(ucavailability.DefinirDiaInput{
			ProviderID: "provider-1", Data: data, Tipo: availability.TipoExtra,
			Blocos: []ucavailability.BlocoInput{{InicioMinutos: 720, FimMinutos: 480}},
		})
		if err != availability.ErrFimAntesDoInicio {
			t.Errorf("esperava ErrFimAntesDoInicio, got: %v", err)
		}
	})

	t.Run("retorna erro quando extra vem sem blocos", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoDefinirDiaUseCase(repo)

		_, err := uc.Executar(ucavailability.DefinirDiaInput{
			ProviderID: "provider-1", Data: data, Tipo: availability.TipoExtra,
		})
		if err != availability.ErrExtraSemBlocos {
			t.Errorf("esperava ErrExtraSemBlocos, got: %v", err)
		}
	})
}
