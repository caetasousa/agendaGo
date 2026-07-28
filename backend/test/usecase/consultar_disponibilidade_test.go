package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/domain/availability"
	ucavailability "agendago/internal/usecase/availability"
	"agendago/test/repository/memoria"
)

func novoAmbienteDisponibilidade(t *testing.T, aceitaAgendamentos bool) (*ucavailability.ConsultarDisponibilidadeUseCase, *memoria.AvailabilityMemoria) {
	t.Helper()
	usuarios, membros, providers := fakesDePrestador()
	_, p := criarPrestador(usuarios, membros, providers, "provider-1", "João Silva", "joao@email.com", "11999998888", senhaDeTeste)
	if aceitaAgendamentos {
		p.AtivarAgenda()
	}
	providers.Salvar(p)

	availRepo := memoria.NovoAvailabilityMemoria()
	uc := ucavailability.NovoConsultarDisponibilidadeUseCase(availRepo, providers, membros)
	return uc, availRepo
}

func TestConsultarDisponibilidade(t *testing.T) {
	// 2026-08-10 é uma segunda-feira; 2026-08-15 é um sábado.
	segunda := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	sabado := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

	t.Run("bloqueio deixa o dia vazio mesmo sendo dia útil", func(t *testing.T) {
		uc, availRepo := novoAmbienteDisponibilidade(t, true)
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

	t.Run("extra substitui o expediente padrão pelos blocos da definição", func(t *testing.T) {
		uc, availRepo := novoAmbienteDisponibilidade(t, true)
		blocoExtra, _ := availability.NovoTimeBlock(600, 660)
		excecao, _ := availability.NovaDateException("exc-1", "provider-1", segunda, availability.TipoExtra, []availability.TimeBlock{blocoExtra})
		availRepo.SalvarExcecao(excecao)

		blocos, err := uc.Executar(ucavailability.ConsultarDisponibilidadeInput{ProviderID: "provider-1", Data: segunda})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(blocos) != 1 || blocos[0].InicioMinutos != 600 {
			t.Errorf("esperava blocos da definição extra, got: %v", blocos)
		}
	})

	t.Run("dia útil sem definição própria: aplica o expediente padrão sugerido", func(t *testing.T) {
		uc, _ := novoAmbienteDisponibilidade(t, true)

		blocos, err := uc.Executar(ucavailability.ConsultarDisponibilidadeInput{ProviderID: "provider-1", Data: segunda})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(blocos) != 2 || blocos[0].InicioMinutos != 480 || blocos[1].InicioMinutos != 840 {
			t.Errorf("esperava expediente padrão (08-12, 14-18), got: %v", blocos)
		}
	})

	t.Run("dia útil sem definição própria: aplica o expediente padrão configurado pelo prestador", func(t *testing.T) {
		usuarios, membros, providers := fakesDePrestador()
		_, p := criarPrestador(usuarios, membros, providers, "provider-1", "João Silva", "joao@email.com", "11999998888", senhaDeTeste)
		p.AtivarAgenda()
		bloco, _ := availability.NovoTimeBlock(9*60, 11*60)
		p.DefinirHorariosPadrao([]availability.TimeBlock{bloco})
		providers.Salvar(p)

		availRepo := memoria.NovoAvailabilityMemoria()
		uc := ucavailability.NovoConsultarDisponibilidadeUseCase(availRepo, providers, membros)

		blocos, err := uc.Executar(ucavailability.ConsultarDisponibilidadeInput{ProviderID: "provider-1", Data: segunda})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(blocos) != 1 || blocos[0].InicioMinutos != 9*60 || blocos[0].FimMinutos != 11*60 {
			t.Errorf("esperava o bloco configurado 09-11, got: %v", blocos)
		}
	})

	t.Run("fim de semana sem definição própria: sem expediente padrão", func(t *testing.T) {
		uc, _ := novoAmbienteDisponibilidade(t, true)

		blocos, err := uc.Executar(ucavailability.ConsultarDisponibilidadeInput{ProviderID: "provider-1", Data: sabado})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(blocos) != 0 {
			t.Errorf("esperava vazio no fim de semana, got: %v", blocos)
		}
	})

	t.Run("extra no fim de semana vale mesmo sem expediente padrão", func(t *testing.T) {
		uc, availRepo := novoAmbienteDisponibilidade(t, true)
		bloco, _ := availability.NovoTimeBlock(480, 720)
		excecao, _ := availability.NovaDateException("exc-1", "provider-1", sabado, availability.TipoExtra, []availability.TimeBlock{bloco})
		availRepo.SalvarExcecao(excecao)

		blocos, err := uc.Executar(ucavailability.ConsultarDisponibilidadeInput{ProviderID: "provider-1", Data: sabado})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(blocos) != 1 || blocos[0].InicioMinutos != 480 {
			t.Errorf("esperava bloco da definição extra, got: %v", blocos)
		}
	})

	t.Run("AceitaAgendamentos=false: nunca aplica expediente padrão", func(t *testing.T) {
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
		availRepo := memoria.NovoAvailabilityMemoria()
		_, membros, providers := fakesDePrestador()
		uc := ucavailability.NovoConsultarDisponibilidadeUseCase(availRepo, providers, membros)

		_, err := uc.Executar(ucavailability.ConsultarDisponibilidadeInput{ProviderID: "id-fantasma", Data: segunda})
		if err != ucavailability.ErrProviderNaoEncontrado {
			t.Errorf("esperava ErrProviderNaoEncontrado, got: %v", err)
		}
	})
}
