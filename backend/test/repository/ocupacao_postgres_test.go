//go:build integration

package repository_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/appointment"
	"agendago/internal/domain/client"
	"agendago/internal/domain/ocupacao"
	"agendago/internal/domain/provider"
)

func TestOcupacaoPostgres(t *testing.T) {
	pool := novoPool(t)
	providers := repository.NovoProviderPostgres(pool)
	repo := repository.NovoOcupacaoPostgres(pool)

	const idProvider = "eeeeeeee-0000-0000-0000-000000000001"
	p, _ := provider.Novo(idProvider, "Agenda com compromissos")
	if err := providers.Salvar(p); err != nil {
		t.Fatalf("preparar prestador: %v", err)
	}

	data := time.Date(2027, 3, 15, 0, 0, 0, 0, time.UTC)

	t.Run("salva e lista por período", func(t *testing.T) {
		o, _ := ocupacao.Nova("11111111-aaaa-0000-0000-000000000001", idProvider, data, 9*60, 10*60, "Médico", ocupacao.OrigemManual)
		if err := repo.Salvar(o); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		achados, err := repo.ListarPorPeriodo(idProvider, data, data)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(achados) != 1 {
			t.Fatalf("esperava 1 compromisso, got: %d", len(achados))
		}
		if achados[0].Titulo != "Médico" {
			t.Errorf("esperava título 'Médico', got: %q", achados[0].Titulo)
		}
		if achados[0].Origem != ocupacao.OrigemManual {
			t.Errorf("esperava origem manual, got: %s", achados[0].Origem)
		}
	})

	t.Run("título vazio volta vazio, não quebra na leitura", func(t *testing.T) {
		// A coluna é nullable e o domínio trabalha com string: se o Scan não
		// tratar o NULL, a leitura estoura.
		o, _ := ocupacao.Nova("11111111-aaaa-0000-0000-000000000002", idProvider, data, 11*60, 12*60, "", ocupacao.OrigemManual)
		if err := repo.Salvar(o); err != nil {
			t.Fatalf("esperava sucesso ao salvar sem título, got: %v", err)
		}
		lido, err := repo.BuscarPorID(o.ID)
		if err != nil {
			t.Fatalf("esperava sucesso na leitura, got: %v", err)
		}
		if lido == nil || lido.Titulo != "" {
			t.Errorf("esperava título vazio, got: %+v", lido)
		}
	})

	t.Run("fora do período não aparece", func(t *testing.T) {
		outroDia := data.AddDate(0, 0, 10)
		achados, err := repo.ListarPorPeriodo(idProvider, outroDia, outroDia)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(achados) != 0 {
			t.Errorf("esperava nenhum compromisso em outro dia, got: %d", len(achados))
		}
	})

	t.Run("remover devolve o horário", func(t *testing.T) {
		o, _ := ocupacao.Nova("11111111-aaaa-0000-0000-000000000003", idProvider, data.AddDate(0, 0, 1), 9*60, 10*60, "", ocupacao.OrigemManual)
		repo.Salvar(o)
		if err := repo.Remover(o.ID); err != nil {
			t.Fatalf("esperava sucesso ao remover, got: %v", err)
		}
		lido, _ := repo.BuscarPorID(o.ID)
		if lido != nil {
			t.Error("esperava o compromisso removido")
		}
	})
}

// O bug que este teste existe para impedir: o slot deixa de ser OFERTADO
// quando há compromisso, mas nada impedia um POST direto na rota de
// solicitação com aquele horário. Sem o segundo EXISTS no SalvarSeLivre, o
// cliente reserva por cima do compromisso pessoal do prestador.
func TestSalvarSeLivreRespeitaOcupacao(t *testing.T) {
	pool := novoPool(t)
	providers := repository.NovoProviderPostgres(pool)
	clients := repository.NovoClientPostgres(pool)
	ocupacoes := repository.NovoOcupacaoPostgres(pool)
	appointments := repository.NovoAppointmentPostgres(pool)

	const (
		idProvider = "eeeeeeee-0000-0000-0000-000000000002"
		idCliente  = "ffffffff-0000-0000-0000-000000000002"
	)
	p, _ := provider.Novo(idProvider, "Agenda protegida")
	if err := providers.Salvar(p); err != nil {
		t.Fatalf("preparar prestador: %v", err)
	}
	c, _ := client.NovoComConta(idCliente, "Cliente da Ocupação", "cliente.ocupacao@email.com", "hash")
	if err := clients.Salvar(c); err != nil {
		t.Fatalf("preparar cliente: %v", err)
	}

	data := time.Date(2027, 4, 20, 0, 0, 0, 0, time.UTC)
	agora := time.Date(2027, 4, 1, 8, 0, 0, 0, time.UTC)

	o, _ := ocupacao.Nova("11111111-bbbb-0000-0000-000000000001", idProvider, data, 14*60, 15*60, "Dentista", ocupacao.OrigemManual)
	if err := ocupacoes.Salvar(o); err != nil {
		t.Fatalf("preparar compromisso: %v", err)
	}

	t.Run("reserva em cima do compromisso é recusada", func(t *testing.T) {
		a, err := appointment.Novo("aaaaaaaa-bbbb-0000-0000-000000000001", idProvider, c.ID, data, 14*60, 15*60, agora, 24*time.Hour)
		if err != nil {
			t.Fatalf("montar agendamento: %v", err)
		}
		if err := appointments.SalvarSeLivre(a, agora); err != appointment.ErrConflitoHorario {
			t.Errorf("esperava ErrConflitoHorario, got: %v", err)
		}
	})

	t.Run("reserva que só encosta no compromisso é aceita", func(t *testing.T) {
		// Sobreposição real é o critério, não o buffer: o buffer é regra de
		// oferta. Recusar aqui seria mais restritivo do que a própria oferta.
		a, err := appointment.Novo("aaaaaaaa-bbbb-0000-0000-000000000002", idProvider, c.ID, data, 15*60, 16*60, agora, 24*time.Hour)
		if err != nil {
			t.Fatalf("montar agendamento: %v", err)
		}
		if err := appointments.SalvarSeLivre(a, agora); err != nil {
			t.Errorf("esperava sucesso em horário adjacente, got: %v", err)
		}
	})

	t.Run("compromisso de outro dia não bloqueia", func(t *testing.T) {
		outroDia := data.AddDate(0, 0, 1)
		a, err := appointment.Novo("aaaaaaaa-bbbb-0000-0000-000000000003", idProvider, c.ID, outroDia, 14*60, 15*60, agora, 24*time.Hour)
		if err != nil {
			t.Fatalf("montar agendamento: %v", err)
		}
		if err := appointments.SalvarSeLivre(a, agora); err != nil {
			t.Errorf("esperava sucesso em outro dia, got: %v", err)
		}
	})
}
