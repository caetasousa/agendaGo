//go:build integration

package repository_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/cancellation"
)

func TestCancellationPostgres(t *testing.T) {
	repo := repository.NovoCancellationPostgres(novoPool(t))

	t.Run("salva e busca token por hash, sem apagar na leitura", func(t *testing.T) {
		tok := cancellation.Novo("hash-aaa", "88888888-8888-8888-8888-888888888888", time.Hour)
		if err := repo.Salvar(tok); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		encontrado, err := repo.BuscarPorTokenHash("hash-aaa")
		if err != nil {
			t.Fatalf("esperava sucesso na busca, got: %v", err)
		}
		if encontrado == nil {
			t.Fatal("esperava encontrar o token salvo")
		}
		if encontrado.AppointmentID != "88888888-8888-8888-8888-888888888888" {
			t.Errorf("appointmentID inesperado: %s", encontrado.AppointmentID)
		}

		// a leitura não consome — o token continua disponível para o
		// cancelamento subsequente; Remover é quem invalida (uso único)
		denovo, err := repo.BuscarPorTokenHash("hash-aaa")
		if err != nil || denovo == nil {
			t.Errorf("esperava o token ainda presente na segunda leitura, got: %v (%v)", denovo, err)
		}
	})

	t.Run("retorna (nil, nil) quando o hash não existe", func(t *testing.T) {
		encontrado, err := repo.BuscarPorTokenHash("hash-inexistente")
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if encontrado != nil {
			t.Errorf("esperava nil, got: %v", encontrado)
		}
	})

	t.Run("Remover apaga o token, invalidando-o para reuso", func(t *testing.T) {
		tok := cancellation.Novo("hash-remover", "99999999-9999-9999-9999-999999999999", time.Hour)
		if err := repo.Salvar(tok); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		if err := repo.Remover("hash-remover"); err != nil {
			t.Fatalf("esperava sucesso ao remover, got: %v", err)
		}
		if ainda, _ := repo.BuscarPorTokenHash("hash-remover"); ainda != nil {
			t.Error("esperava o token removido")
		}
	})

	t.Run("RemoverExpirados apaga só os tokens vencidos", func(t *testing.T) {
		expirado := cancellation.Novo("hash-cancel-expirado", "77777777-7777-7777-7777-777777777777", -time.Hour)
		valido := cancellation.Novo("hash-cancel-valido", "66666666-6666-6666-6666-666666666666", time.Hour)
		if err := repo.Salvar(expirado); err != nil {
			t.Fatalf("esperava sucesso ao salvar expirado, got: %v", err)
		}
		if err := repo.Salvar(valido); err != nil {
			t.Fatalf("esperava sucesso ao salvar válido, got: %v", err)
		}

		if err := repo.RemoverExpirados(); err != nil {
			t.Fatalf("esperava sucesso na limpeza, got: %v", err)
		}

		if ainda, _ := repo.BuscarPorTokenHash("hash-cancel-expirado"); ainda != nil {
			t.Error("esperava o token expirado removido")
		}
		if sumiu, _ := repo.BuscarPorTokenHash("hash-cancel-valido"); sumiu == nil {
			t.Error("esperava o token válido preservado")
		}
	})
}
