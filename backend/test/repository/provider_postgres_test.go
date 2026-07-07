//go:build integration

// Testes de integração do repositório Postgres. Sobem um PostgreSQL efêmero
// via Testcontainers e rodam contra o banco real. Executar com:
//
//	go test -tags=integration ./...
package repository_test

import (
	"context"
	"path/filepath"
	"sort"
	"testing"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/provider"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// migrationsOrdenadas devolve os caminhos absolutos de todas as migrations em
// ordem, para aplicá-las em sequência no banco de teste.
func migrationsOrdenadas(t *testing.T) []string {
	t.Helper()
	caminhos, err := filepath.Glob("../../migrations/V*.sql")
	if err != nil {
		t.Fatalf("resolver caminhos das migrations: %v", err)
	}
	sort.Strings(caminhos)

	absolutos := make([]string, len(caminhos))
	for i, c := range caminhos {
		abs, err := filepath.Abs(c)
		if err != nil {
			t.Fatalf("resolver caminho absoluto da migration %s: %v", c, err)
		}
		absolutos[i] = abs
	}
	return absolutos
}

// novoPool sobe um Postgres efêmero com todas as migrations aplicadas e
// devolve um pool pronto para uso. O container é encerrado automaticamente
// no fim do teste.
func novoPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("agendago_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.WithInitScripts(migrationsOrdenadas(t)...),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp"),
		),
	)
	if err != nil {
		t.Fatalf("subir container postgres: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("obter connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("criar pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

func TestProviderPostgres(t *testing.T) {
	repo := repository.NovoProviderPostgres(novoPool(t))

	t.Run("salva e busca prestador por email", func(t *testing.T) {
		p, _ := provider.Novo("11111111-1111-1111-1111-111111111111", "João Silva", "joao@email.com", "12345678")
		if err := repo.Salvar(p); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		encontrado, err := repo.BuscarPorEmail("joao@email.com")
		if err != nil {
			t.Fatalf("esperava sucesso na busca, got: %v", err)
		}
		if encontrado == nil {
			t.Fatal("esperava encontrar o prestador salvo")
		}
		if encontrado.ID != p.ID {
			t.Errorf("esperava ID %s, got: %s", p.ID, encontrado.ID)
		}
		if encontrado.Nome != "João Silva" {
			t.Errorf("esperava nome 'João Silva', got: %s", encontrado.Nome)
		}
		if encontrado.CriadoEm.IsZero() {
			t.Error("criado_em deveria vir preenchido pelo banco")
		}
	})

	t.Run("retorna (nil, nil) quando email não existe", func(t *testing.T) {
		encontrado, err := repo.BuscarPorEmail("inexistente@email.com")
		if err != nil {
			t.Fatalf("não esperava erro para email inexistente, got: %v", err)
		}
		if encontrado != nil {
			t.Errorf("esperava nil para email inexistente, got: %v", encontrado)
		}
	})

	t.Run("falha ao salvar email duplicado (constraint UNIQUE)", func(t *testing.T) {
		p1, _ := provider.Novo("22222222-2222-2222-2222-222222222222", "Ana", "ana@email.com", "12345678")
		p2, _ := provider.Novo("33333333-3333-3333-3333-333333333333", "Ana Duplicada", "ana@email.com", "12345678")

		if err := repo.Salvar(p1); err != nil {
			t.Fatalf("esperava sucesso no primeiro salvar, got: %v", err)
		}
		if err := repo.Salvar(p2); err == nil {
			t.Error("esperava erro ao salvar email duplicado")
		}
	})

	t.Run("salva e busca prestador por ID", func(t *testing.T) {
		p, _ := provider.Novo("88888888-8888-8888-8888-888888888888", "Carlos Souza", "carlos@email.com", "12345678")
		if err := repo.Salvar(p); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		encontrado, err := repo.BuscarPorID(p.ID)
		if err != nil {
			t.Fatalf("esperava sucesso na busca, got: %v", err)
		}
		if encontrado == nil {
			t.Fatal("esperava encontrar o prestador salvo")
		}
		if encontrado.Email != "carlos@email.com" {
			t.Errorf("esperava email 'carlos@email.com', got: %s", encontrado.Email)
		}
	})

	t.Run("retorna (nil, nil) quando ID não existe", func(t *testing.T) {
		encontrado, err := repo.BuscarPorID("99999999-9999-9999-9999-999999999999")
		if err != nil {
			t.Fatalf("não esperava erro para ID inexistente, got: %v", err)
		}
		if encontrado != nil {
			t.Errorf("esperava nil para ID inexistente, got: %v", encontrado)
		}
	})

	t.Run("Atualizar persiste as preferências e reflete no BuscarPorID", func(t *testing.T) {
		p, _ := provider.Novo("44444444-4444-4444-4444-444444444444", "Diana Prince", "diana@email.com", "12345678")
		if err := repo.Salvar(p); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		p.AtivarAgenda()
		if err := p.DefinirDescanso(20); err != nil {
			t.Fatalf("esperava sucesso ao definir descanso, got: %v", err)
		}
		if err := repo.Atualizar(p); err != nil {
			t.Fatalf("esperava sucesso ao atualizar, got: %v", err)
		}

		encontrado, err := repo.BuscarPorID(p.ID)
		if err != nil {
			t.Fatalf("esperava sucesso na busca, got: %v", err)
		}
		if !encontrado.AceitaAgendamentos {
			t.Error("esperava AceitaAgendamentos true após Atualizar")
		}
		if encontrado.DescansoMinutos != 20 {
			t.Errorf("esperava DescansoMinutos 20, got: %d", encontrado.DescansoMinutos)
		}
		if !encontrado.AtualizadoEm.After(encontrado.CriadoEm) {
			t.Error("esperava AtualizadoEm posterior a CriadoEm após Atualizar")
		}
	})
}
