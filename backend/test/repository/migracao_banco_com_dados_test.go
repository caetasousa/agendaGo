//go:build integration

// A V14 vai rodar sobre o banco de produção que já existe, não sobre um banco
// vazio. Este teste semeia o estado de um servidor em uso — prestador com
// expediente, cliente, agendamento confirmado e sessão aberta — aplica a
// migration e verifica que nada disso se perde. É a resposta para "preciso
// apagar o banco antes de implantar?".
package repository_test

import (
	"context"
	"os"
	"testing"
)

func TestMigracaoV14SobreBancoComDados(t *testing.T) {
	ctx := context.Background()
	pool := novoPoolAte(t, 13)

	const (
		idPrestador = "11111111-aaaa-bbbb-cccc-000000000001"
		idCliente   = "22222222-aaaa-bbbb-cccc-000000000002"
		idAgenda    = "33333333-aaaa-bbbb-cccc-000000000003"
		emailPrest  = "prestador.producao@email.com"
		senhaPrest  = "$argon2id$v=19$m=19456,t=2,p=1$hashficticiodeproducao$naoimporta"
		tokenSessao = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	)

	semear := []struct {
		oque string
		sql  string
		args []any
	}{
		{"prestador", `
			INSERT INTO providers (id, nome, email, telefone, senha_hash, ativo,
				aceita_agendamentos, descanso_minutos, duracao_atendimento_minutos)
			VALUES ($1, 'Marina Fisioterapeuta', $2, '11999998888', $3, TRUE, TRUE, 10, 50)`,
			[]any{idPrestador, emailPrest, senhaPrest}},

		{"expediente do prestador", `
			INSERT INTO horarios_padrao (id, provider_id, inicio_minutos, fim_minutos)
			VALUES (gen_random_uuid(), $1, 480, 720)`,
			[]any{idPrestador}},

		{"cliente", `
			INSERT INTO clients (id, nome, email, telefone, senha_hash, ativo)
			VALUES ($1, 'João Cliente', 'joao.cliente@email.com', '11888887777', 'hash', TRUE)`,
			[]any{idCliente}},

		{"agendamento confirmado", `
			INSERT INTO appointments (id, provider_id, client_id, data, inicio_minutos, fim_minutos, status, expira_em)
			VALUES ($1, $2, $3, CURRENT_DATE + 3, 540, 590, 'CONFIRMADO', NOW() + INTERVAL '3 days')`,
			[]any{idAgenda, idPrestador, idCliente}},

		{"sessão aberta do prestador", `
			INSERT INTO sessions (token_hash, user_id, user_type, expira_em)
			VALUES ($1, $2, 'provider', NOW() + INTERVAL '7 days')`,
			[]any{tokenSessao, idPrestador}},
	}
	for _, s := range semear {
		if _, err := pool.Exec(ctx, s.sql, s.args...); err != nil {
			t.Fatalf("semear %s: %v", s.oque, err)
		}
	}

	sql, err := os.ReadFile("../../migrations/V14__separa_identidade_de_provider.sql")
	if err != nil {
		t.Fatalf("ler a migration V14: %v", err)
	}
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("aplicar a V14 sobre banco com dados: %v", err)
	}

	t.Run("a credencial do prestador sobrevive", func(t *testing.T) {
		var email, senha string
		var ativo bool
		err := pool.QueryRow(ctx,
			`SELECT email, senha_hash, ativo FROM usuarios WHERE id = $1`, idPrestador).
			Scan(&email, &senha, &ativo)
		if err != nil {
			t.Fatalf("esperava a conta do prestador migrada, got: %v", err)
		}
		// Se o hash mudar, a senha antiga para de funcionar e o prestador fica
		// trancado fora da própria conta.
		if email != emailPrest || senha != senhaPrest || !ativo {
			t.Errorf("esperava credencial intacta, got: email=%s ativo=%v", email, ativo)
		}
	})

	t.Run("a sessão aberta continua resolvendo", func(t *testing.T) {
		var vale bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM sessions s
				JOIN usuarios u ON u.id = s.user_id
				JOIN provider_membros m ON m.usuario_id = u.id
				WHERE s.token_hash = $1 AND s.user_type = 'provider' AND m.papel = 'dono'
			)`, tokenSessao).Scan(&vale)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if !vale {
			t.Error("esperava a sessão anterior ao deploy continuar válida e ligada à agenda")
		}
	})

	t.Run("o agendamento continua ligado à agenda certa", func(t *testing.T) {
		var nomePrestador, nomeCliente, status string
		err := pool.QueryRow(ctx, `
			SELECT p.nome, c.nome, a.status
			FROM appointments a
			JOIN providers p ON p.id = a.provider_id
			JOIN clients c ON c.id = a.client_id
			WHERE a.id = $1`, idAgenda).Scan(&nomePrestador, &nomeCliente, &status)
		if err != nil {
			t.Fatalf("esperava o agendamento intacto, got: %v", err)
		}
		if nomePrestador != "Marina Fisioterapeuta" || nomeCliente != "João Cliente" || status != "CONFIRMADO" {
			t.Errorf("esperava o agendamento preservado, got: %s / %s / %s", nomePrestador, nomeCliente, status)
		}
	})

	t.Run("o expediente continua na agenda", func(t *testing.T) {
		var blocos int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM horarios_padrao WHERE provider_id = $1`, idPrestador).Scan(&blocos); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if blocos != 1 {
			t.Errorf("esperava o expediente preservado, got: %d blocos", blocos)
		}
	})

	t.Run("o cliente não é tocado pela migração", func(t *testing.T) {
		var email string
		var ativo bool
		if err := pool.QueryRow(ctx,
			`SELECT email, ativo FROM clients WHERE id = $1`, idCliente).Scan(&email, &ativo); err != nil {
			t.Fatalf("esperava o cliente intacto, got: %v", err)
		}
		if email != "joao.cliente@email.com" || !ativo {
			t.Errorf("esperava o cliente preservado, got: %s ativo=%v", email, ativo)
		}
		// clients segue fora do modelo de identidade — decisão registrada em
		// docs/regra-de-negocio.md.
		var virouUsuario bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM usuarios WHERE id = $1)`, idCliente).Scan(&virouUsuario); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if virouUsuario {
			t.Error("cliente não deveria virar conta de usuario")
		}
	})
}
