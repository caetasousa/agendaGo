package usecase_test

import (
	"fmt"
	"testing"
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/domain/provider"
	"agendago/internal/pkg/paging"
	ucappointment "agendago/internal/usecase/appointment"
	ucprovider "agendago/internal/usecase/provider"
	"agendago/test/repository/memoria"
)

// As listagens do sistema crescem para sempre (vitrine, histórico de
// agendamentos). Estes testes fixam o contrato que impede uma delas de virar
// "traga o banco inteiro": a resposta traz no máximo o limite pedido, o total
// informa quanto existe, e o offset caminha sem repetir nem pular item.

func TestPaginacaoDaVitrine(t *testing.T) {
	repo := memoria.NovoProviderMemoria()
	for i := 0; i < 5; i++ {
		// nome com índice para a ordenação ser previsível (ordem alfabética)
		p, _ := provider.Novo(fmt.Sprintf("p-%d", i), fmt.Sprintf("Prestador %d", i), fmt.Sprintf("p%d@email.com", i), "11999998888", "hash")
		p.AtivarAgenda()
		repo.Salvar(p)
	}
	banido, _ := provider.Novo("p-banido", "Prestador Banido", "banido@email.com", "11999998888", "hash")
	banido.Banir()
	repo.Salvar(banido)

	uc := ucprovider.NovoListarUseCase(repo)

	t.Run("respeita o limite e informa o total de ativos", func(t *testing.T) {
		out, err := uc.Executar(paging.Normalizar(2, 0))
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Prestadores) != 2 {
			t.Errorf("esperava 2 prestadores na página, got: %d", len(out.Prestadores))
		}
		// o banido não entra na conta: a vitrine é só de ativos
		if out.Total != 5 {
			t.Errorf("esperava total 5 (banido fora), got: %d", out.Total)
		}
	})

	t.Run("offset caminha sem repetir nem pular prestador", func(t *testing.T) {
		vistos := map[string]bool{}
		for offset := 0; offset < 5; offset += 2 {
			out, err := uc.Executar(paging.Normalizar(2, offset))
			if err != nil {
				t.Fatalf("offset %d: %v", offset, err)
			}
			for _, p := range out.Prestadores {
				if vistos[p.ID] {
					t.Errorf("prestador %s apareceu em duas páginas", p.ID)
				}
				vistos[p.ID] = true
			}
		}
		if len(vistos) != 5 {
			t.Errorf("esperava percorrer os 5 ativos, got: %d", len(vistos))
		}
	})

	t.Run("limite acima do teto é reduzido ao máximo permitido", func(t *testing.T) {
		if pag := paging.Normalizar(10_000, 0); pag.Limite != paging.LimiteMaximo {
			t.Errorf("esperava limite reduzido a %d, got: %d", paging.LimiteMaximo, pag.Limite)
		}
	})
}

func TestPaginacaoDosAgendamentos(t *testing.T) {
	repo := memoria.NovoAppointmentMemoria()
	providerRepo := memoria.NovoProviderMemoria()
	clientRepo := memoria.NovoClientMemoria()

	base := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	agora := base.AddDate(0, 0, -1)
	for i := 0; i < 5; i++ {
		a, err := appointment.Novo(
			fmt.Sprintf("a-%d", i), "provider-1", "client-1",
			base.AddDate(0, 0, i), 9*60, 10*60, agora, time.Hour,
		)
		if err != nil {
			t.Fatalf("esperava criar agendamento, got: %v", err)
		}
		repo.SalvarSeLivre(a, agora)
	}

	uc := ucappointment.NovoListarUseCase(repo, providerRepo, clientRepo)

	t.Run("primeira página traz os mais recentes e o total completo", func(t *testing.T) {
		out, err := uc.DoPrestador(ucappointment.ListarInput{
			UsuarioID: "provider-1",
			Pagina:    paging.Normalizar(2, 0),
			Agora:     agora,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Agendamentos) != 2 {
			t.Fatalf("esperava 2 agendamentos na página, got: %d", len(out.Agendamentos))
		}
		if out.Total != 5 {
			t.Errorf("esperava total 5, got: %d", out.Total)
		}
		// ordem decrescente: a data mais distante primeiro, o histórico depois
		if !out.Agendamentos[0].Data.After(out.Agendamentos[1].Data) {
			t.Errorf("esperava ordem decrescente por data, got: %v antes de %v",
				out.Agendamentos[0].Data, out.Agendamentos[1].Data)
		}
	})

	t.Run("offset além do fim devolve página vazia, não erro", func(t *testing.T) {
		out, err := uc.DoPrestador(ucappointment.ListarInput{
			UsuarioID: "provider-1",
			Pagina:    paging.Normalizar(2, 99),
			Agora:     agora,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Agendamentos) != 0 {
			t.Errorf("esperava página vazia, got: %d", len(out.Agendamentos))
		}
		if out.Total != 5 {
			t.Errorf("esperava total 5 mesmo na página vazia, got: %d", out.Total)
		}
	})
}
