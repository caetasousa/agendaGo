package usecase_test

import (
	"testing"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/availability"
	ucavailability "agendago/internal/usecase/availability"
)

func TestConsultarGradeSemanal(t *testing.T) {
	t.Run("prestador sem grade configurada devolve Dias vazio", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		uc := ucavailability.NovoConsultarGradeSemanalUseCase(repo)

		out, err := uc.Executar(ucavailability.ConsultarGradeSemanalInput{ProviderID: "provider-1"})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Dias) != 0 {
			t.Errorf("esperava Dias vazio, got: %v", out.Dias)
		}
	})

	t.Run("devolve a grade configurada", func(t *testing.T) {
		repo := repository.NovoAvailabilityMemoria()
		definir := ucavailability.NovoDefinirGradeSemanalUseCase(repo)
		definir.Executar(ucavailability.DefinirGradeSemanalInput{
			ProviderID: "provider-1",
			BlocosPorDia: map[availability.DiaSemana][]ucavailability.BlocoInput{
				availability.Quarta: {{InicioMinutos: 480, FimMinutos: 720}},
			},
		})

		uc := ucavailability.NovoConsultarGradeSemanalUseCase(repo)
		out, err := uc.Executar(ucavailability.ConsultarGradeSemanalInput{ProviderID: "provider-1"})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Dias[availability.Quarta]) != 1 {
			t.Errorf("esperava 1 bloco na quarta, got: %v", out.Dias)
		}
	})
}
