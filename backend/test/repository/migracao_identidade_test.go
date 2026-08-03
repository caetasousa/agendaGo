//go:build integration

// Teste da migração V14, que separa identidade (usuarios) de agenda
// (providers). A V14 não converte prestador antigo nenhum — o banco é
// descartável neste momento do projeto, e converter exigiria decidir em SQL o
// que é do domínio. Sobra verificar o que ela de fato promete: a identidade
// sai de `providers` de verdade, e as tabelas novas aceitam o formato que o
// código passou a escrever.
package repository_test

import (
	"context"
	"os"
	"testing"
)

func TestMigracaoV14SeparaIdentidade(t *testing.T) {
	ctx := context.Background()
	// Para na V13: o estado do banco antes desta fase.
	pool := novoPoolAte(t, 13)

	sql, err := os.ReadFile("../../migrations/V14__separa_identidade_de_provider.sql")
	if err != nil {
		t.Fatalf("ler a migration V14: %v", err)
	}
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("aplicar a V14: %v", err)
	}

	t.Run("identidade sai de providers", func(t *testing.T) {
		// Enquanto as colunas existirem NOT NULL, o código novo — que parou de
		// escrevê-las — não consegue inserir prestador nenhum. Removê-las é o
		// que fecha a separação; se ficassem, seriam duas fontes da verdade.
		var restantes int
		err := pool.QueryRow(ctx, `
			SELECT count(*) FROM information_schema.columns
			WHERE table_name = 'providers'
			  AND column_name IN ('email', 'senha_hash', 'telefone', 'ativo')`).Scan(&restantes)
		if err != nil {
			t.Fatalf("consultar colunas de providers: %v", err)
		}
		if restantes != 0 {
			t.Errorf("esperava nenhuma coluna de identidade em providers, got: %d", restantes)
		}
	})

	t.Run("prestador novo entra sem informar identidade", func(t *testing.T) {
		_, err := pool.Exec(ctx, `
			INSERT INTO providers (id, nome, aceita_agendamentos, descanso_minutos,
				duracao_atendimento_minutos, permite_marcacao_pelo_prestador)
			VALUES ($1, $2, TRUE, 0, 60, FALSE)`,
			"cccccccc-9999-9999-9999-999999999999", "Prestador Novo")
		if err != nil {
			t.Errorf("esperava inserir prestador só com dados de agenda, got: %v", err)
		}
	})

	t.Run("a conta e o vínculo aceitam o que o código escreve", func(t *testing.T) {
		// A V14 cria as duas tabelas mas não as popula. Quem popula é o
		// domínio, e este é o formato que ele manda — inclusive o papel, que
		// vive em internal/domain/membro e não no schema.
		const idUsuario = "cccccccc-8888-8888-8888-888888888888"
		_, err := pool.Exec(ctx, `
			INSERT INTO usuarios (id, email, senha_hash, telefone, ativo)
			VALUES ($1, 'dono@email.com', 'hash', '11999998888', TRUE)`, idUsuario)
		if err != nil {
			t.Fatalf("esperava inserir conta, got: %v", err)
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO provider_membros (id, usuario_id, provider_id, papel)
			VALUES (gen_random_uuid(), $1, $2, 'dono')`,
			idUsuario, "cccccccc-9999-9999-9999-999999999999")
		if err != nil {
			t.Errorf("esperava ligar a conta à agenda como dono, got: %v", err)
		}
	})
}
