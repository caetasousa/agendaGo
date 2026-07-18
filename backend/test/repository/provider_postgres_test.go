//go:build integration

// Testes de integração do repositório Postgres. Sobem um PostgreSQL efêmero
// via Testcontainers e rodam contra o banco real. Executar com:
//
//	go test -tags=integration ./...
package repository_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/availability"
	"agendago/internal/domain/provider"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// migrationsOrdenadas copia as migrations para um diretório temporário com um
// prefixo numérico com zero-padding (001_, 002_, …) e devolve esses caminhos.
//
// O Postgres executa os init scripts em ordem alfabética do NOME do arquivo
// dentro do container — não na ordem em que os passamos. Com os nomes
// originais (V1, V10, V11, V2…), "V10" viria antes de "V1" e um ALTER TABLE
// rodaria antes do CREATE. O prefixo com zero-padding faz a ordem alfabética
// coincidir com a ordem de versão. (O Flyway real, no compose, já ordena por
// versão numérica — o problema é só do runner de teste.)
func migrationsOrdenadas(t *testing.T) []string {
	t.Helper()
	caminhos, err := filepath.Glob("../../migrations/V*.sql")
	if err != nil {
		t.Fatalf("resolver caminhos das migrations: %v", err)
	}
	sort.Slice(caminhos, func(i, j int) bool {
		return versaoMigration(t, caminhos[i]) < versaoMigration(t, caminhos[j])
	})

	dir := t.TempDir()
	prefixados := make([]string, len(caminhos))
	for i, c := range caminhos {
		conteudo, err := os.ReadFile(c)
		if err != nil {
			t.Fatalf("ler migration %s: %v", c, err)
		}
		destino := filepath.Join(dir, fmt.Sprintf("%03d_%s", i, filepath.Base(c)))
		if err := os.WriteFile(destino, conteudo, 0o600); err != nil {
			t.Fatalf("copiar migration %s: %v", c, err)
		}
		prefixados[i] = destino
	}
	return prefixados
}

// versaoMigration extrai o número N do nome de arquivo "V{N}__descricao.sql".
func versaoMigration(t *testing.T, caminho string) int {
	t.Helper()
	base := filepath.Base(caminho)
	fim := strings.Index(base, "__")
	if fim < 0 || !strings.HasPrefix(base, "V") {
		t.Fatalf("nome de migration inesperado: %s", base)
	}
	n, err := strconv.Atoi(base[1:fim])
	if err != nil {
		t.Fatalf("versão de migration inválida em %s: %v", base, err)
	}
	return n
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
		// BasicWaitStrategies espera o log "ready to accept connections"
		// aparecer duas vezes (o Postgres reinicia após o primeiro startup) e
		// só então a porta ser servida. Esperar só a porta pega o servidor no
		// meio do restart e causa "connection reset by peer" intermitente.
		tcpostgres.BasicWaitStrategies(),
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
		p, _ := provider.Novo("11111111-1111-1111-1111-111111111111", "João Silva", "joao@email.com", "11999998888", "12345678")
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
		if len(encontrado.HorariosPadrao) != 2 {
			t.Errorf("esperava 2 blocos do expediente sugerido, got: %v", encontrado.HorariosPadrao)
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
		p1, _ := provider.Novo("22222222-2222-2222-2222-222222222222", "Ana", "ana@email.com", "11999998888", "12345678")
		p2, _ := provider.Novo("33333333-3333-3333-3333-333333333333", "Ana Duplicada", "ana@email.com", "11999998888", "12345678")

		if err := repo.Salvar(p1); err != nil {
			t.Fatalf("esperava sucesso no primeiro salvar, got: %v", err)
		}
		if err := repo.Salvar(p2); err == nil {
			t.Error("esperava erro ao salvar email duplicado")
		}
	})

	t.Run("salva e busca prestador por ID", func(t *testing.T) {
		p, _ := provider.Novo("88888888-8888-8888-8888-888888888888", "Carlos Souza", "carlos@email.com", "11999998888", "12345678")
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
		if !encontrado.PermiteMarcacaoPeloPrestador {
			t.Error("esperava PermiteMarcacaoPeloPrestador true por padrão")
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
		p, _ := provider.Novo("44444444-4444-4444-4444-444444444444", "Diana Prince", "diana@email.com", "11999998888", "12345678")
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

	t.Run("Atualizar persiste PermiteMarcacaoPeloPrestador desativado", func(t *testing.T) {
		p, _ := provider.Novo("33333333-3333-3333-3333-333333333333", "Fábio Lima", "fabio@email.com", "11999998888", "12345678")
		if err := repo.Salvar(p); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		p.DesativarMarcacaoPeloPrestador()
		if err := repo.Atualizar(p); err != nil {
			t.Fatalf("esperava sucesso ao atualizar, got: %v", err)
		}

		encontrado, err := repo.BuscarPorID(p.ID)
		if err != nil {
			t.Fatalf("esperava sucesso na busca, got: %v", err)
		}
		if encontrado.PermiteMarcacaoPeloPrestador {
			t.Error("esperava PermiteMarcacaoPeloPrestador false após desativar e Atualizar")
		}
	})

	t.Run("Atualizar substitui o expediente padrão (delete-all + insert)", func(t *testing.T) {
		p, _ := provider.Novo("55555555-5555-5555-5555-555555555555", "Eva Souza", "eva@email.com", "11999998888", "12345678")
		if err := repo.Salvar(p); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		bloco1, _ := availability.NovoTimeBlock(9*60, 11*60)
		bloco2, _ := availability.NovoTimeBlock(13*60, 15*60)
		bloco3, _ := availability.NovoTimeBlock(16*60, 18*60)
		if err := p.DefinirHorariosPadrao([]availability.TimeBlock{bloco1, bloco2, bloco3}); err != nil {
			t.Fatalf("esperava sucesso ao definir horários, got: %v", err)
		}
		if err := repo.Atualizar(p); err != nil {
			t.Fatalf("esperava sucesso ao atualizar, got: %v", err)
		}

		encontrado, err := repo.BuscarPorID(p.ID)
		if err != nil {
			t.Fatalf("esperava sucesso na busca, got: %v", err)
		}
		if len(encontrado.HorariosPadrao) != 3 {
			t.Fatalf("esperava 3 blocos, got: %d", len(encontrado.HorariosPadrao))
		}
		if encontrado.HorariosPadrao[0].InicioMinutos != 9*60 {
			t.Errorf("esperava blocos ordenados por início, got: %v", encontrado.HorariosPadrao)
		}

		// atualizar de novo com lista vazia deve remover os blocos anteriores
		if err := encontrado.DefinirHorariosPadrao(nil); err != nil {
			t.Fatalf("esperava sucesso ao limpar horários, got: %v", err)
		}
		if err := repo.Atualizar(encontrado); err != nil {
			t.Fatalf("esperava sucesso ao atualizar, got: %v", err)
		}
		semHorarios, _ := repo.BuscarPorID(p.ID)
		if len(semHorarios.HorariosPadrao) != 0 {
			t.Errorf("esperava nenhum bloco após limpar, got: %v", semHorarios.HorariosPadrao)
		}
	})
}
