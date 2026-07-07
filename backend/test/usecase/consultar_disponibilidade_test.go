package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/availability"
	"agendago/internal/domain/provider"
	ucavailability "agendago/internal/usecase/availability"
)

func novoAmbienteDisponibilidade(t *testing.T, aceitaAgendamentos bool) (*ucavailability.ConsultarDisponibilidadeUseCase, *repository.AvailabilityMemoria) {
	t.Helper()
	providerRepo := repository.NovoProviderMemoria()
	p, _ := provider.Novo("provider-1", "João Silva", "joao@email.com", "hash-da-senha")
	if aceitaAgendamentos {
		p.AtivarAgenda()
	}
	providerRepo.Salvar(p)

	availRepo := repository.NovoAvailabilityMemoria()
	uc := ucavailability.NovoConsultarDisponibilidadeUseCase(availRepo, availRepo, providerRepo)
	return uc, availRepo
}

func TestConsultarDisponibilidade(t *testing.T) {
	// 2026-08-10 é uma segunda-feira; 2026-08-15 é um sábado.
	segunda := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	sabado := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

	t.Run("bloqueio ignora grade semanal configurada para o dia", func(t *testing.T) {
		uc, availRepo := novoAmbienteDisponibilidade(t, true)
		bloco, _ := availability.NovoTimeBlock(480, 720)
		s, _ := availability.NovaWeeklySchedule("provider-1", map[availability.DiaSemana][]availability.TimeBlock{
			availability.Segunda: {bloco},
		})
		availRepo.Salvar(s)
		excecao, _ := availability.NovaDateException("exc-1", "provider-1", segunda, availability.TipoBloqueio, nil)
		availRepo.SalvarExcecao(excecao)

		blocos, err := uc.Executar(ucavailability.ConsultarDisponibilidadeInput{ProviderID: "provider-1", Data: segunda})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(blocos) != 0 {
			t.Errorf("esperava vazio (bloqueio), got: %v", blocos)
		}
	})

	t.Run("extra ignora grade semanal e devolve os blocos da exceção", func(t *testing.T) {
		uc, availRepo := novoAmbienteDisponibilidade(t, true)
		blocoGrade, _ := availability.NovoTimeBlock(480, 720)
		s, _ := availability.NovaWeeklySchedule("provider-1", map[availability.DiaSemana][]availability.TimeBlock{
			availability.Segunda: {blocoGrade},
		})
		availRepo.Salvar(s)
		blocoExtra, _ := availability.NovoTimeBlock(600, 660)
		excecao, _ := availability.NovaDateException("exc-1", "provider-1", segunda, availability.TipoExtra, []availability.TimeBlock{blocoExtra})
		availRepo.SalvarExcecao(excecao)

		blocos, err := uc.Executar(ucavailability.ConsultarDisponibilidadeInput{ProviderID: "provider-1", Data: segunda})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(blocos) != 1 || blocos[0].InicioMinutos != 600 {
			t.Errorf("esperava blocos da exceção extra, got: %v", blocos)
		}
	})

	t.Run("sem exceção, usa a grade semanal configurada", func(t *testing.T) {
		uc, availRepo := novoAmbienteDisponibilidade(t, true)
		bloco, _ := availability.NovoTimeBlock(480, 720)
		s, _ := availability.NovaWeeklySchedule("provider-1", map[availability.DiaSemana][]availability.TimeBlock{
			availability.Segunda: {bloco},
		})
		availRepo.Salvar(s)

		blocos, err := uc.Executar(ucavailability.ConsultarDisponibilidadeInput{ProviderID: "provider-1", Data: segunda})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(blocos) != 1 || blocos[0].InicioMinutos != 480 {
			t.Errorf("esperava bloco da grade semanal, got: %v", blocos)
		}
	})

	t.Run("grade configurada mas dia específico vazio não cai no default", func(t *testing.T) {
		uc, availRepo := novoAmbienteDisponibilidade(t, true)
		bloco, _ := availability.NovoTimeBlock(480, 720)
		s, _ := availability.NovaWeeklySchedule("provider-1", map[availability.DiaSemana][]availability.TimeBlock{
			availability.Terca: {bloco}, // segunda fica de fora, de propósito
		})
		availRepo.Salvar(s)

		blocos, err := uc.Executar(ucavailability.ConsultarDisponibilidadeInput{ProviderID: "provider-1", Data: segunda})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(blocos) != 0 {
			t.Errorf("esperava vazio (dia configurado sem expediente), got: %v", blocos)
		}
	})

	t.Run("nunca configurou grade, aceita agendamentos, dia útil: aplica default comercial", func(t *testing.T) {
		uc, _ := novoAmbienteDisponibilidade(t, true)

		blocos, err := uc.Executar(ucavailability.ConsultarDisponibilidadeInput{ProviderID: "provider-1", Data: segunda})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(blocos) != 2 || blocos[0].InicioMinutos != 480 || blocos[1].InicioMinutos != 840 {
			t.Errorf("esperava default comercial (08-12, 14-18), got: %v", blocos)
		}
	})

	t.Run("nunca configurou grade, fim de semana: sem default", func(t *testing.T) {
		uc, _ := novoAmbienteDisponibilidade(t, true)

		blocos, err := uc.Executar(ucavailability.ConsultarDisponibilidadeInput{ProviderID: "provider-1", Data: sabado})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(blocos) != 0 {
			t.Errorf("esperava vazio no fim de semana, got: %v", blocos)
		}
	})

	t.Run("nunca configurou grade, AceitaAgendamentos=false: nunca aplica default", func(t *testing.T) {
		uc, _ := novoAmbienteDisponibilidade(t, false)

		blocos, err := uc.Executar(ucavailability.ConsultarDisponibilidadeInput{ProviderID: "provider-1", Data: segunda})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(blocos) != 0 {
			t.Errorf("esperava vazio (agenda desativada), got: %v", blocos)
		}
	})

	t.Run("retorna erro quando prestador não existe", func(t *testing.T) {
		availRepo := repository.NovoAvailabilityMemoria()
		providerRepo := repository.NovoProviderMemoria()
		uc := ucavailability.NovoConsultarDisponibilidadeUseCase(availRepo, availRepo, providerRepo)

		_, err := uc.Executar(ucavailability.ConsultarDisponibilidadeInput{ProviderID: "id-fantasma", Data: segunda})
		if err != ucavailability.ErrProviderNaoEncontrado {
			t.Errorf("esperava ErrProviderNaoEncontrado, got: %v", err)
		}
	})
}
