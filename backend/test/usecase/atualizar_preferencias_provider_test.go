package usecase_test

import (
	"testing"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/provider"
	ucprovider "agendago/internal/usecase/provider"
)

func novoProviderComPreferencias(repo *repository.ProviderMemoria) *provider.Provider {
	p, _ := provider.Novo("provider-1", "João Silva", "joao@email.com", "hash-da-senha")
	repo.Salvar(p)
	return p
}

func TestAtualizarPreferenciasProvider(t *testing.T) {
	t.Run("ativa a agenda e define o descanso", func(t *testing.T) {
		repo := repository.NovoProviderMemoria()
		novoProviderComPreferencias(repo)
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(repo)

		out, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			ProviderID:         "provider-1",
			AceitaAgendamentos: true,
			DescansoMinutos:    15,
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
			ProviderID:         "provider-1",
			AceitaAgendamentos: false,
			DescansoMinutos:    0,
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
			ProviderID:         "provider-1",
			AceitaAgendamentos: true,
			DescansoMinutos:    -1,
		})
		if err != provider.ErrDescansoInvalido {
			t.Errorf("esperava ErrDescansoInvalido, got: %v", err)
		}
	})

	t.Run("retorna erro quando prestador não existe", func(t *testing.T) {
		repo := repository.NovoProviderMemoria()
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(repo)

		_, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			ProviderID:         "id-inexistente",
			AceitaAgendamentos: true,
			DescansoMinutos:    0,
		})
		if err != ucprovider.ErrProviderNaoEncontrado {
			t.Errorf("esperava ErrProviderNaoEncontrado, got: %v", err)
		}
	})
}
