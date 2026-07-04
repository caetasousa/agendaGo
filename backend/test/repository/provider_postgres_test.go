//go:build integration

// Testes de integração do repositório Postgres. Sobem um PostgreSQL efêmero
// via Testcontainers e rodam contra o banco real. Executar com:
//
//	go test -tags=integration ./...
package repository_test

import (
	"context"
	"path/filepath"
	"testing"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/provider"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// novoPool sobe um Postgres efêmero com a migration aplicada e devolve um pool
// pronto para uso. O container é encerrado automaticamente no fim do teste.
func novoPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	migration, err := filepath.Abs("../../migrations/V1__cria_tabela_providers.sql")
	if err != nil {
		t.Fatalf("resolver caminho da migration: %v", err)
	}

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("agendago_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.WithInitScripts(migration),
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
}
