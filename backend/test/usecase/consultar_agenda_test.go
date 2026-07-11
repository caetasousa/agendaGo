package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/availability"
	"agendago/internal/domain/provider"
	ucavailability "agendago/internal/usecase/availability"
)

func novoAmbienteAgenda(t *testing.T, aceitaAgendamentos bool) (*ucavailability.ConsultarAgendaUseCase, *repository.AvailabilityMemoria) {
	t.Helper()
	providerRepo := repository.NovoProviderMemoria()
	p, _ := provider.Novo("provider-1", "João Silva", "joao@email.com", "hash-da-senha")
	if aceitaAgendamentos {
		p.AtivarAgenda()
	}
	providerRepo.Salvar(p)

	availRepo := repository.NovoAvailabilityMemoria()
	uc := ucavailability.NovoConsultarAgendaUseCase(availRepo, providerRepo)
	return uc, availRepo
}

func TestConsultarAgenda(t *testing.T) {
	// Semana de 2026-08-10 (segunda) a 2026-08-16 (domingo).
	segunda := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	domingo := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)

	t.Run("resolve a semana inteira: dias úteis com padrão, fim de semana vazio", func(t *testing.T) {
		uc, _ := novoAmbienteAgenda(t, true)

		out, err := uc.Executar(ucavailability.ConsultarAgendaInput{ProviderID: "provider-1", De: segunda, Ate: domingo})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Dias) != 7 {
			t.Fatalf("esperava 7 dias, got: %d", len(out.Dias))
		}
		if !out.AceitaAgendamentos {
			t.Error("esperava AceitaAgendamentos=true")
		}
		if out.Dias[0].Origem != ucavailability.OrigemPadrao || len(out.Dias[0].Blocos) != 2 {
			t.Errorf("esperava segunda com padrão comercial, got: %+v", out.Dias[0])
		}
		sabado := out.Dias[5]
		if sabado.Origem != ucavailability.OrigemPadrao || len(sabado.Blocos) != 0 {
			t.Errorf("esperava sábado padrão sem blocos, got: %+v", sabado)
		}
	})

	t.Run("definições próprias sobrepõem o padrão", func(t *testing.T) {
		uc, availRepo := novoAmbienteAgenda(t, true)
		bloqueio, _ := availability.NovaDateException("exc-1", "provider-1", segunda, availability.TipoBloqueio, nil)
		availRepo.SalvarExcecao(bloqueio)
		bloco, _ := availability.NovoTimeBlock(600, 660)
		extra, _ := availability.NovaDateException("exc-2", "provider-1", domingo, availability.TipoExtra, []availability.TimeBlock{bloco})
		availRepo.SalvarExcecao(extra)

		out, err := uc.Executar(ucavailability.ConsultarAgendaInput{ProviderID: "provider-1", De: segunda, Ate: domingo})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.Dias[0].Origem != ucavailability.OrigemBloqueio || len(out.Dias[0].Blocos) != 0 {
			t.Errorf("esperava segunda bloqueada, got: %+v", out.Dias[0])
		}
		if out.Dias[6].Origem != ucavailability.OrigemExtra || len(out.Dias[6].Blocos) != 1 {
			t.Errorf("esperava domingo com horários personalizados, got: %+v", out.Dias[6])
		}
	})

	t.Run("agenda desativada: dias padrão saem sem blocos", func(t *testing.T) {
		uc, _ := novoAmbienteAgenda(t, false)

		out, err := uc.Executar(ucavailability.ConsultarAgendaInput{ProviderID: "provider-1", De: segunda, Ate: domingo})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.AceitaAgendamentos {
			t.Error("esperava AceitaAgendamentos=false")
		}
		for _, d := range out.Dias {
			if len(d.Blocos) != 0 {
				t.Errorf("esperava dia sem blocos com agenda desativada, got: %+v", d)
			}
		}
	})

	t.Run("retorna ErrPeriodoInvalido para período invertido", func(t *testing.T) {
		uc, _ := novoAmbienteAgenda(t, true)

		_, err := uc.Executar(ucavailability.ConsultarAgendaInput{ProviderID: "provider-1", De: domingo, Ate: segunda})
		if err != ucavailability.ErrPeriodoInvalido {
			t.Errorf("esperava ErrPeriodoInvalido, got: %v", err)
		}
	})

	t.Run("retorna ErrPeriodoInvalido para período longo demais", func(t *testing.T) {
		uc, _ := novoAmbienteAgenda(t, true)

		_, err := uc.Executar(ucavailability.ConsultarAgendaInput{
			ProviderID: "provider-1", De: segunda, Ate: segunda.AddDate(1, 0, 0),
		})
		if err != ucavailability.ErrPeriodoInvalido {
			t.Errorf("esperava ErrPeriodoInvalido, got: %v", err)
		}
	})

	t.Run("retorna erro quando prestador não existe", func(t *testing.T) {
		availRepo := repository.NovoAvailabilityMemoria()
		providerRepo := repository.NovoProviderMemoria()
		uc := ucavailability.NovoConsultarAgendaUseCase(availRepo, providerRepo)

		_, err := uc.Executar(ucavailability.ConsultarAgendaInput{ProviderID: "id-fantasma", De: segunda, Ate: domingo})
		if err != ucavailability.ErrProviderNaoEncontrado {
			t.Errorf("esperava ErrProviderNaoEncontrado, got: %v", err)
		}
	})
}
