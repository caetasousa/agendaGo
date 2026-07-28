//go:build integration

// Teste da migração V14, que separa identidade (usuarios) de agenda
// (providers). Verifica as duas metades da mudança: a conversão não perde dado
// de prestador que já exista, e a identidade some de `providers` de verdade.
package repository_test

import (
	"context"
	"os"
	"testing"
)

func TestMigracaoV14SeparaIdentidade(t *testing.T) {
	ctx := context.Background()
	// Para na V13: o estado de um banco de produção antes desta fase.
	pool := novoPoolAte(t, 13)

	const (
		idProvider = "aaaaaaaa-1111-2222-3333-444444444444"
		email      = "prestador.antigo@email.com"
		senhaHash  = "$2a$10$hashficticiodetestecomtamanhosuficiente"
		telefone   = "11999998888"
		// sessions.token_hash é CHAR(64): um valor mais curto seria preenchido
		// com espaços pelo Postgres e deixaria o teste dependendo disso.
		tokenHash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	)

	_, err := pool.Exec(ctx, `
		INSERT INTO providers (
			id, nome, email, telefone, senha_hash, ativo,
			aceita_agendamentos, descanso_minutos, duracao_atendimento_minutos
		) VALUES ($1, $2, $3, $4, $5, TRUE, TRUE, 10, 50)`,
		idProvider, "Prestador Antigo", email, telefone, senhaHash)
	if err != nil {
		t.Fatalf("semear prestador no estado pré-V14: %v", err)
	}

	// Uma sessão aberta antes da migração. Ela aponta para o id do prestador —
	// é exatamente o que a V14 não pode invalidar.
	_, err = pool.Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, user_type, expira_em)
		VALUES ($1, $2, 'provider', NOW() + INTERVAL '1 day')`,
		tokenHash, idProvider)
	if err != nil {
		t.Fatalf("semear sessão aberta: %v", err)
	}

	sql, err := os.ReadFile("../../migrations/V14__separa_identidade_de_provider.sql")
	if err != nil {
		t.Fatalf("ler a migration V14: %v", err)
	}
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("aplicar a V14: %v", err)
	}

	t.Run("a identidade é copiada sem perder nada", func(t *testing.T) {
		var (
			id, emailNovo, senhaNova, telefoneNovo string
			ativo                                  bool
		)
		err := pool.QueryRow(ctx, `
			SELECT id, email, senha_hash, telefone, ativo FROM usuarios WHERE id = $1`,
			idProvider).Scan(&id, &emailNovo, &senhaNova, &telefoneNovo, &ativo)
		if err != nil {
			t.Fatalf("esperava um usuário para o prestador migrado, got: %v", err)
		}
		if emailNovo != email {
			t.Errorf("esperava email %s, got: %s", email, emailNovo)
		}
		// O hash é o que faz a senha antiga continuar valendo. Se ele mudar, o
		// prestador não loga mais.
		if senhaNova != senhaHash {
			t.Errorf("esperava senha_hash preservado, got: %s", senhaNova)
		}
		if telefoneNovo != telefone {
			t.Errorf("esperava telefone %s, got: %s", telefone, telefoneNovo)
		}
		if !ativo {
			t.Error("esperava o usuário ativo, como o prestador era")
		}
	})

	t.Run("o id do usuário é o mesmo do provider", func(t *testing.T) {
		// É esta igualdade que mantém sessions.user_id e
		// social_identities.user_id válidos sem migrá-las.
		var iguais bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM usuarios u
				JOIN providers p ON p.id = u.id
				JOIN sessions s ON s.user_id = u.id AND s.user_type = 'provider'
			)`).Scan(&iguais)
		if err != nil {
			t.Fatalf("esperava sucesso na verificação, got: %v", err)
		}
		if !iguais {
			t.Error("esperava a sessão aberta continuar resolvendo para o usuário migrado")
		}
	})

	t.Run("todo prestador vira dono da própria agenda", func(t *testing.T) {
		var usuarioID, providerID, papel string
		err := pool.QueryRow(ctx, `
			SELECT usuario_id, provider_id, papel FROM provider_membros WHERE provider_id = $1`,
			idProvider).Scan(&usuarioID, &providerID, &papel)
		if err != nil {
			t.Fatalf("esperava um vínculo para o prestador migrado, got: %v", err)
		}
		if papel != "dono" {
			t.Errorf("esperava papel 'dono', got: %s", papel)
		}
		if usuarioID != idProvider || providerID != idProvider {
			t.Errorf("esperava vínculo do prestador consigo mesmo, got: usuario=%s provider=%s", usuarioID, providerID)
		}
	})

	t.Run("um vínculo por prestador, nem mais nem menos", func(t *testing.T) {
		var prestadores, donos int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM providers`).Scan(&prestadores); err != nil {
			t.Fatalf("contar prestadores: %v", err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM provider_membros WHERE papel = 'dono'`).Scan(&donos); err != nil {
			t.Fatalf("contar donos: %v", err)
		}
		if prestadores != donos {
			t.Errorf("esperava um dono por prestador, got: %d prestadores e %d donos", prestadores, donos)
		}
	})

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
			INSERT INTO providers (id, nome, aceita_agendamentos, descanso_minutos, duracao_atendimento_minutos)
			VALUES ($1, $2, TRUE, 0, 60)`,
			"cccccccc-9999-9999-9999-999999999999", "Prestador Novo")
		if err != nil {
			t.Errorf("esperava inserir prestador só com dados de agenda, got: %v", err)
		}
	})
}
