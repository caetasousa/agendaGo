package usecase_test

import (
	"testing"

	ucprovider "agendago/internal/usecase/provider"
	"agendago/test/repository/memoria"
)

func TestVitrine(t *testing.T) {
	// ambiente com um prestador ativo ofertando, um com agenda fechada e um banido
	novoRepo := func(t *testing.T) (*memoria.ProviderMemoria, *memoria.MembroMemoria) {
		t.Helper()
		usuarios, membros, providers := fakesDePrestador()

		_, ativo := criarPrestador(usuarios, membros, providers, "p-ativo", "Ana Ativa", "ana@email.com", "11999998888", "hash")
		ativo.AtivarAgenda()
		providers.Salvar(ativo)

		criarPrestador(usuarios, membros, providers, "p-fechado", "Fabio Fechado", "fabio@email.com", "11999998888", "hash")

		// Banido: a agenda oferta, mas a CONTA está banida — é o banimento que
		// tem que ganhar, e ele mora em usuarios desde a V14.
		uBanido, banido := criarPrestador(usuarios, membros, providers, "p-banido", "Bruno Banido", "bruno@email.com", "11999998888", "hash")
		banido.AtivarAgenda()
		providers.Salvar(banido)
		uBanido.Banir()
		usuarios.Atualizar(uBanido)

		return providers, membros
	}

	t.Run("lista só prestadores não banidos, com o status da agenda", func(t *testing.T) {
		uc := func() *ucprovider.ListarUseCase {
			providers, _ := novoRepo(t)
			return ucprovider.NovoListarUseCase(providers)
		}()

		out, err := uc.Executar(paginaPadrao)
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
		uc := func() *ucprovider.BuscarResumoUseCase {
			providers, membros := novoRepo(t)
			return ucprovider.NovoBuscarResumoUseCase(providers, membros)
		}()

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
		uc := func() *ucprovider.BuscarResumoUseCase {
			providers, membros := novoRepo(t)
			return ucprovider.NovoBuscarResumoUseCase(providers, membros)
		}()

		resumo, err := uc.Executar("p-banido")
		if err != nil {
			t.Fatalf("esperava sucesso (não vaza banimento como 404), got: %v", err)
		}
		if resumo.AceitaAgendamentos {
			t.Error("banido não deveria constar como ofertando horários")
		}
	})

	t.Run("id inexistente retorna ErrProviderNaoEncontrado", func(t *testing.T) {
		uc := func() *ucprovider.BuscarResumoUseCase {
			providers, membros := novoRepo(t)
			return ucprovider.NovoBuscarResumoUseCase(providers, membros)
		}()
		if _, err := uc.Executar("fantasma"); err != ucprovider.ErrProviderNaoEncontrado {
			t.Errorf("esperava ErrProviderNaoEncontrado, got: %v", err)
		}
	})
}
