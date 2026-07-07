package usecase_test

import (
	"testing"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/availability"
	ucavailability "agendago/internal/usecase/availability"
)

func TestDefinirGradeSemanal(t *testing.T) {
	t.Run("salva grade válida e reflete no repositório", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoDefinirGradeSemanalUseCase(repo)

		out, err := uc.Executar(ucavailability.DefinirGradeSemanalInput{
			ProviderID: "provider-1",
			BlocosPorDia: map[availability.DiaSemana][]ucavailability.BlocoInput{
				availability.Segunda: {{InicioMinutos: 480, FimMinutos: 720}},
			},
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Dias[availability.Segunda]) != 1 {
			t.Fatalf("esperava 1 bloco na segunda, got: %v", out.Dias)
		}

		salva, _ := repo.Buscar("provider-1")
		if salva == nil {
			t.Fatal("esperava grade persistida")
		}
	})

	t.Run("propaga erro de bloco inválido", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoDefinirGradeSemanalUseCase(repo)

		_, err := uc.Executar(ucavailability.DefinirGradeSemanalInput{
			ProviderID: "provider-1",
			BlocosPorDia: map[availability.DiaSemana][]ucavailability.BlocoInput{
				availability.Segunda: {{InicioMinutos: 485, FimMinutos: 720}},
			},
		})
		if err != availability.ErrGranularidadeInvalida {
			t.Errorf("esperava ErrGranularidadeInvalida, got: %v", err)
		}
	})

	t.Run("propaga erro de overlap franco", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoDefinirGradeSemanalUseCase(repo)

		_, err := uc.Executar(ucavailability.DefinirGradeSemanalInput{
			ProviderID: "provider-1",
			BlocosPorDia: map[availability.DiaSemana][]ucavailability.BlocoInput{
				availability.Segunda: {
					{InicioMinutos: 480, FimMinutos: 780},
					{InicioMinutos: 720, FimMinutos: 840},
				},
			},
		})
		if err != availability.ErrBlocosSobrepostos {
			t.Errorf("esperava ErrBlocosSobrepostos, got: %v", err)
		}
	})

	t.Run("salvar substitui a grade anterior por completo", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoDefinirGradeSemanalUseCase(repo)

		uc.Executar(ucavailability.DefinirGradeSemanalInput{
			ProviderID: "provider-1",
			BlocosPorDia: map[availability.DiaSemana][]ucavailability.BlocoInput{
				availability.Segunda: {{InicioMinutos: 480, FimMinutos: 720}},
			},
		})

		out, err := uc.Executar(ucavailability.DefinirGradeSemanalInput{
			ProviderID: "provider-1",
			BlocosPorDia: map[availability.DiaSemana][]ucavailability.BlocoInput{
				availability.Terca: {{InicioMinutos: 540, FimMinutos: 600}},
			},
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Dias[availability.Segunda]) != 0 {
			t.Error("esperava que segunda-feira não tivesse mais blocos")
		}
		if len(out.Dias[availability.Terca]) != 1 {
			t.Error("esperava 1 bloco em terça-feira")
		}
	})
}
