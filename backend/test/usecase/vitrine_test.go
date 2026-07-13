package usecase_test

import (
	"testing"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/provider"
	ucprovider "agendago/internal/usecase/provider"
)

func TestVitrine(t *testing.T) {
	// ambiente com um prestador ativo ofertando, um com agenda fechada e um banido
	novoRepo := func(t *testing.T) *repository.ProviderMemoria {
		t.Helper()
		repo := repository.NovoProviderMemoria()

		ativo, _ := provider.Novo("p-ativo", "Ana Ativa", "ana@email.com", "11999998888", "hash")
		ativo.AtivarAgenda()
		repo.Salvar(ativo)

		fechado, _ := provider.Novo("p-fechado", "Fabio Fechado", "fabio@email.com", "11999998888", "hash")
		repo.Salvar(fechado)

		banido, _ := provider.Novo("p-banido", "Bruno Banido", "bruno@email.com", "11999998888", "hash")
		banido.AtivarAgenda()
		banido.Banir()
		repo.Salvar(banido)

		return repo
	}

	t.Run("lista só prestadores não banidos, com o status da agenda", func(t *testing.T) {
		uc := ucprovider.NovoListarUseCase(novoRepo(t))

		out, err := uc.Executar()
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Prestadores) != 2 {
			t.Fatalf("esperava 2 prestadores na vitrine (banido fora), got: %d", len(out.Prestadores))
		}
		porID := map[string]ucprovider.PrestadorResumo{}
		for _, p := range out.Prestadores {
			porID[p.ID] = p
		}
		if _, ok := porID["p-banido"]; ok {
			t.Error("prestador banido não deveria aparecer na vitrine")
		}
		if !porID["p-ativo"].AceitaAgendamentos {
			t.Error("esperava prestador ativo ofertando horários")
		}
		if porID["p-fechado"].AceitaAgendamentos {
			t.Error("prestador com agenda fechada não deveria constar como ofertando")
		}
	})

	t.Run("busca o resumo público pelo id, com a duração do atendimento", func(t *testing.T) {
		uc := ucprovider.NovoBuscarResumoUseCase(novoRepo(t))

		resumo, err := uc.Executar("p-ativo")
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if resumo.Nome != "Ana Ativa" || !resumo.AceitaAgendamentos {
			t.Errorf("esperava resumo do prestador ativo, got: %+v", resumo)
		}
		if resumo.DuracaoAtendimentoMinutos <= 0 {
			t.Errorf("esperava duração de atendimento preenchida, got: %d", resumo.DuracaoAtendimentoMinutos)
		}
	})

	t.Run("banido no link direto aparece como não ofertando, sem vazar o motivo", func(t *testing.T) {
		uc := ucprovider.NovoBuscarResumoUseCase(novoRepo(t))

		resumo, err := uc.Executar("p-banido")
		if err != nil {
			t.Fatalf("esperava sucesso (não vaza banimento como 404), got: %v", err)
		}
		if resumo.AceitaAgendamentos {
			t.Error("banido não deveria constar como ofertando horários")
		}
	})

	t.Run("id inexistente retorna ErrProviderNaoEncontrado", func(t *testing.T) {
		uc := ucprovider.NovoBuscarResumoUseCase(novoRepo(t))
		if _, err := uc.Executar("fantasma"); err != ucprovider.ErrProviderNaoEncontrado {
			t.Errorf("esperava ErrProviderNaoEncontrado, got: %v", err)
		}
	})
}
