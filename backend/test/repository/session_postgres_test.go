//go:build integration

package repository_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/session"
)

func TestSessionPostgres(t *testing.T) {
	repo := repository.NovoSessionPostgres(novoPool(t))

	t.Run("salva e busca sessão por token hash", func(t *testing.T) {
		s := session.Nova("hash-aaa", "88888888-8888-8888-8888-888888888888", session.TipoProvider, time.Hour)
		if err := repo.Salvar(s); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		encontrada, err := repo.BuscarPorTokenHash("hash-aaa")
		if err != nil {
			t.Fatalf("esperava sucesso na busca, got: %v", err)
		}
		if encontrada == nil {
			t.Fatal("esperava encontrar a sessão salva")
		}
		if encontrada.UserType != session.TipoProvider {
			t.Errorf("esperava TipoProvider, got: %s", encontrada.UserType)
		}
	})

	t.Run("retorna (nil, nil) quando token hash não existe", func(t *testing.T) {
		encontrada, err := repo.BuscarPorTokenHash("hash-inexistente")
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if encontrada != nil {
			t.Errorf("esperava nil, got: %v", encontrada)
		}
	})

	t.Run("remove sessão pelo token hash", func(t *testing.T) {
		s := session.Nova("hash-bbb", "99999999-9999-9999-9999-999999999999", session.TipoClient, time.Hour)
		repo.Salvar(s)

		if err := repo.Remover("hash-bbb"); err != nil {
			t.Fatalf("esperava sucesso ao remover, got: %v", err)
		}

		encontrada, _ := repo.BuscarPorTokenHash("hash-bbb")
		if encontrada != nil {
			t.Error("esperava sessão removida")
		}
	})

	t.Run("remover token hash inexistente não é erro", func(t *testing.T) {
		if err := repo.Remover("hash-nunca-existiu"); err != nil {
			t.Errorf("não esperava erro, got: %v", err)
		}
	})

	t.Run("remove sessões expiradas", func(t *testing.T) {
		expirada := session.Nova("hash-expirada", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", session.TipoProvider, -time.Hour)
		valida := session.Nova("hash-valida", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", session.TipoProvider, time.Hour)
		repo.Salvar(expirada)
		repo.Salvar(valida)

		if err := repo.RemoverExpiradas(); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		encontradaExpirada, _ := repo.BuscarPorTokenHash("hash-expirada")
		if encontradaExpirada != nil {
			t.Error("esperava sessão expirada removida")
		}

		encontradaValida, _ := repo.BuscarPorTokenHash("hash-valida")
		if encontradaValida == nil {
			t.Error("esperava sessão válida mantida")
		}
	})
}
