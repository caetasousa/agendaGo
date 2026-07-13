//go:build integration

package repository_test

import (
	"testing"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/cancellation"
)

func TestCancellationPostgres(t *testing.T) {
	repo := repository.NovoCancellationPostgres(novoPool(t))

	t.Run("salva e busca token por hash, sem apagar na leitura", func(t *testing.T) {
		tok := cancellation.Novo("hash-aaa", "88888888-8888-8888-8888-888888888888")
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

		// diferente do reset de senha: a leitura não consome — o token
		// continua disponível para o cancelamento subsequente
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
}
