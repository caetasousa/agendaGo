//go:build integration

package repository_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/passwordreset"
	"agendago/internal/domain/session"
)

func TestPasswordResetPostgres(t *testing.T) {
	repo := repository.NovoPasswordResetPostgres(novoPool(t))

	t.Run("salva e consome um token, apagando-o", func(t *testing.T) {
		tok := passwordreset.Novo("hash-aaa", "88888888-8888-8888-8888-888888888888", session.TipoProvider, time.Hour)
		if err := repo.Salvar(tok); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		consumido, err := repo.Consumir("hash-aaa")
		if err != nil {
			t.Fatalf("esperava sucesso ao consumir, got: %v", err)
		}
		if consumido == nil {
			t.Fatal("esperava encontrar o token salvo")
		}
		if consumido.UserType != session.TipoProvider {
			t.Errorf("esperava TipoProvider, got: %s", consumido.UserType)
		}

		// consumir de novo devolve nil: o token já foi apagado (uso único)
		segundaVez, err := repo.Consumir("hash-aaa")
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if segundaVez != nil {
			t.Error("esperava nil ao consumir um token já consumido")
		}
	})

	t.Run("retorna (nil, nil) quando o hash não existe", func(t *testing.T) {
		consumido, err := repo.Consumir("hash-inexistente")
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if consumido != nil {
			t.Errorf("esperava nil, got: %v", consumido)
		}
	})

	t.Run("remove todos os tokens de um usuário", func(t *testing.T) {
		alvo := "cccccccc-cccc-cccc-cccc-cccccccccccc"
		outro := "dddddddd-dddd-dddd-dddd-dddddddddddd"
		repo.Salvar(passwordreset.Novo("hash-alvo-1", alvo, session.TipoClient, time.Hour))
		repo.Salvar(passwordreset.Novo("hash-alvo-2", alvo, session.TipoClient, time.Hour))
		repo.Salvar(passwordreset.Novo("hash-outro", outro, session.TipoClient, time.Hour))

		if err := repo.RemoverDoUsuario(alvo); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		if tok, _ := repo.Consumir("hash-alvo-1"); tok != nil {
			t.Error("esperava primeiro token do usuário removido")
		}
		if tok, _ := repo.Consumir("hash-alvo-2"); tok != nil {
			t.Error("esperava segundo token do usuário removido")
		}
		if tok, _ := repo.Consumir("hash-outro"); tok == nil {
			t.Error("token de outro usuário não deveria ser tocado")
		}
	})

	t.Run("remove tokens expirados", func(t *testing.T) {
		expirado := passwordreset.Novo("hash-expirado", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", session.TipoProvider, -time.Hour)
		valido := passwordreset.Novo("hash-valido", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", session.TipoProvider, time.Hour)
		repo.Salvar(expirado)
		repo.Salvar(valido)

		if err := repo.RemoverExpirados(); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		if tok, _ := repo.Consumir("hash-expirado"); tok != nil {
			t.Error("esperava token expirado removido")
		}
		if tok, _ := repo.Consumir("hash-valido"); tok == nil {
			t.Error("esperava token válido mantido")
		}
	})
}
