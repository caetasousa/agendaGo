package usecase_test

import (
	"testing"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/availability"
	"agendago/internal/domain/provider"
	ucprovider "agendago/internal/usecase/provider"
)

func novoProviderComPreferencias(repo *repository.ProviderMemoria) *provider.Provider {
	p, _ := provider.Novo("provider-1", "João Silva", "joao@email.com", "11999998888", "hash-da-senha")
	repo.Salvar(p)
	return p
}

func TestAtualizarPreferenciasProvider(t *testing.T) {
	t.Run("ativa a agenda e define o descanso", func(t *testing.T) {
		repo := repository.NovoProviderMemoria()
		novoProviderComPreferencias(repo)
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(repo)

		out, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			ProviderID:                "provider-1",
			Telefone:                  "11999998888",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        true,
			DescansoMinutos:           15,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if !out.AceitaAgendamentos {
			t.Error("esperava agenda ativada")
		}
		if out.DescansoMinutos != 15 {
			t.Errorf("esperava descanso 15, got: %d", out.DescansoMinutos)
		}

		persistido, _ := repo.BuscarPorID("provider-1")
		if !persistido.AceitaAgendamentos || persistido.DescansoMinutos != 15 {
			t.Error("esperava que as preferências fossem persistidas")
		}
	})

	t.Run("desativa a agenda", func(t *testing.T) {
		repo := repository.NovoProviderMemoria()
		p := novoProviderComPreferencias(repo)
		p.AtivarAgenda()
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(repo)

		out, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			ProviderID:                "provider-1",
			Telefone:                  "11999998888",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        false,
			DescansoMinutos:           0,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.AceitaAgendamentos {
			t.Error("esperava agenda desativada")
		}
	})

	t.Run("retorna erro quando descanso é negativo", func(t *testing.T) {
		repo := repository.NovoProviderMemoria()
		novoProviderComPreferencias(repo)
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(repo)

		_, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			ProviderID:                "provider-1",
			Telefone:                  "11999998888",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        true,
			DescansoMinutos:           -1,
		})
		if err != provider.ErrDescansoInvalido {
			t.Errorf("esperava ErrDescansoInvalido, got: %v", err)
		}
	})

	t.Run("retorna erro quando prestador não existe", func(t *testing.T) {
		repo := repository.NovoProviderMemoria()
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(repo)

		_, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			ProviderID:                "id-inexistente",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        true,
			DescansoMinutos:           0,
		})
		if err != ucprovider.ErrProviderNaoEncontrado {
			t.Errorf("esperava ErrProviderNaoEncontrado, got: %v", err)
		}
	})

	t.Run("define o expediente padrão com três blocos curtos", func(t *testing.T) {
		repo := repository.NovoProviderMemoria()
		novoProviderComPreferencias(repo)
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(repo)

		out, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			ProviderID:                "provider-1",
			Telefone:                  "11999998888",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        true,
			DescansoMinutos:           15,
			HorariosPadrao: []ucprovider.BlocoInput{
				{InicioMinutos: 8 * 60, FimMinutos: 10 * 60},
				{InicioMinutos: 11 * 60, FimMinutos: 13 * 60},
				{InicioMinutos: 15 * 60, FimMinutos: 17 * 60},
			},
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.HorariosPadrao) != 3 {
			t.Fatalf("esperava 3 blocos, got: %d", len(out.HorariosPadrao))
		}

		persistido, _ := repo.BuscarPorID("provider-1")
		if len(persistido.HorariosPadrao) != 3 {
			t.Error("esperava que o expediente padrão fosse persistido")
		}
	})

	t.Run("aceita expediente padrão vazio (nenhum horário)", func(t *testing.T) {
		repo := repository.NovoProviderMemoria()
		novoProviderComPreferencias(repo)
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(repo)

		out, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			ProviderID:                "provider-1",
			Telefone:                  "11999998888",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        false,
			DescansoMinutos:           0,
			HorariosPadrao:            nil,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.HorariosPadrao) != 0 {
			t.Errorf("esperava nenhum bloco, got: %v", out.HorariosPadrao)
		}
	})

	t.Run("retorna erro quando um bloco do expediente padrão é inválido", func(t *testing.T) {
		repo := repository.NovoProviderMemoria()
		novoProviderComPreferencias(repo)
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(repo)

		_, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			ProviderID:                "provider-1",
			Telefone:                  "11999998888",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        true,
			DescansoMinutos:           0,
			HorariosPadrao:            []ucprovider.BlocoInput{{InicioMinutos: 12 * 60, FimMinutos: 8 * 60}},
		})
		if err != availability.ErrFimAntesDoInicio {
			t.Errorf("esperava ErrFimAntesDoInicio, got: %v", err)
		}
	})

	t.Run("retorna erro quando blocos do expediente padrão se sobrepõem", func(t *testing.T) {
		repo := repository.NovoProviderMemoria()
		novoProviderComPreferencias(repo)
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(repo)

		_, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			ProviderID:                "provider-1",
			Telefone:                  "11999998888",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        true,
			DescansoMinutos:           0,
			HorariosPadrao: []ucprovider.BlocoInput{
				{InicioMinutos: 8 * 60, FimMinutos: 13 * 60},
				{InicioMinutos: 12 * 60, FimMinutos: 14 * 60},
			},
		})
		if err != availability.ErrBlocosSobrepostos {
			t.Errorf("esperava ErrBlocosSobrepostos, got: %v", err)
		}
	})
}
