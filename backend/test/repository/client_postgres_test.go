//go:build integration

package repository_test

import (
	"testing"

	"agendago/internal/adapter/repository"
	"agendago/internal/domain/client"
)

func TestClientPostgres(t *testing.T) {
	repo := repository.NovoClientPostgres(novoPool(t))

	t.Run("salva e busca client com conta por email", func(t *testing.T) {
		c, _ := client.NovoComConta("44444444-4444-4444-4444-444444444444", "Maria Silva", "maria@email.com", "hash-da-senha")
		if err := repo.Salvar(c); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		encontrado, err := repo.BuscarPorEmail("maria@email.com")
		if err != nil {
			t.Fatalf("esperava sucesso na busca, got: %v", err)
		}
		if encontrado == nil {
			t.Fatal("esperava encontrar o client salvo")
		}
		if encontrado.SenhaHash != "hash-da-senha" {
			t.Errorf("esperava hash 'hash-da-senha', got: %s", encontrado.SenhaHash)
		}
		if !encontrado.TemConta() {
			t.Error("esperava que o client tivesse conta")
		}
	})

	t.Run("salva e busca client convidado com senha_hash NULL", func(t *testing.T) {
		c, _ := client.NovoConvidado("55555555-5555-5555-5555-555555555555", "Convidado", "convidado@email.com", "11999998888")
		if err := repo.Salvar(c); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		encontrado, err := repo.BuscarPorEmail("convidado@email.com")
		if err != nil {
			t.Fatalf("esperava sucesso na busca, got: %v", err)
		}
		if encontrado == nil {
			t.Fatal("esperava encontrar o client salvo")
		}
		if encontrado.TemConta() {
			t.Error("esperava que o client convidado não tivesse conta")
		}
		if encontrado.SenhaHash != "" {
			t.Errorf("esperava SenhaHash vazio, got: %s", encontrado.SenhaHash)
		}
		if encontrado.Telefone != "11999998888" {
			t.Errorf("esperava o telefone do convidado persistido, got: %s", encontrado.Telefone)
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
		c1, _ := client.NovoConvidado("66666666-6666-6666-6666-666666666666", "Ana", "ana-client@email.com", "11999998888")
		c2, _ := client.NovoConvidado("77777777-7777-7777-7777-777777777777", "Ana Duplicada", "ana-client@email.com", "11999998888")

		if err := repo.Salvar(c1); err != nil {
			t.Fatalf("esperava sucesso no primeiro salvar, got: %v", err)
		}
		if err := repo.Salvar(c2); err == nil {
			t.Error("esperava erro ao salvar email duplicado")
		}
	})

	t.Run("salva e busca client por ID", func(t *testing.T) {
		c, _ := client.NovoComConta("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Bruno Lima", "bruno@email.com", "hash-da-senha")
		if err := repo.Salvar(c); err != nil {
			t.Fatalf("esperava sucesso ao salvar, got: %v", err)
		}

		encontrado, err := repo.BuscarPorID(c.ID)
		if err != nil {
			t.Fatalf("esperava sucesso na busca, got: %v", err)
		}
		if encontrado == nil {
			t.Fatal("esperava encontrar o client salvo")
		}
		if !encontrado.TemConta() {
			t.Error("esperava que o client tivesse conta")
		}
		if encontrado.Email != "bruno@email.com" {
			t.Errorf("esperava email 'bruno@email.com', got: %s", encontrado.Email)
		}
	})

	t.Run("retorna (nil, nil) quando ID não existe", func(t *testing.T) {
		encontrado, err := repo.BuscarPorID("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
		if err != nil {
			t.Fatalf("não esperava erro para ID inexistente, got: %v", err)
		}
		if encontrado != nil {
			t.Errorf("esperava nil para ID inexistente, got: %v", encontrado)
		}
	})

	t.Run("ConverterEmConta transforma convidado em conta preservando o ID", func(t *testing.T) {
		convidado, _ := client.NovoConvidado("77777777-7777-7777-7777-777777777777", "Ex Convidado", "ex-convidado@email.com", "11988887777")
		if err := repo.Salvar(convidado); err != nil {
			t.Fatalf("esperava sucesso ao salvar convidado, got: %v", err)
		}

		if err := repo.ConverterEmConta(convidado.ID, "novo-hash", "11999990000"); err != nil {
			t.Fatalf("esperava sucesso ao converter, got: %v", err)
		}

		encontrado, err := repo.BuscarPorID(convidado.ID)
		if err != nil {
			t.Fatalf("esperava sucesso na busca, got: %v", err)
		}
		if !encontrado.TemConta() {
			t.Error("esperava que o ex-convidado tivesse conta após converter")
		}
		if encontrado.Telefone != "11999990000" {
			t.Errorf("esperava telefone atualizado, got: %s", encontrado.Telefone)
		}
		if encontrado.ID != convidado.ID {
			t.Error("o ID deveria ser preservado na conversão")
		}
	})
}
