package usecase_test

import (
	"testing"

	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	ucprovider "agendago/internal/usecase/provider"
)

func novoUseCase() *ucprovider.CadastrarUseCase {
	return ucprovider.NovoCadastrarUseCase(repository.NovoProviderMemoria(), security.NovoHasherArgon2id())
}

func TestCadastrarProvider(t *testing.T) {
	t.Run("cadastra provider com dados válidos e retorna ID gerado", func(t *testing.T) {
		uc := novoUseCase()
		out, err := uc.Executar(ucprovider.CadastrarInput{
			Nome:  "João Silva",
			Email: "joao@email.com",
			Senha: "12345678",
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.ID == "" {
			t.Error("ID não deve ser vazio")
		}
		if out.Email != "joao@email.com" {
			t.Errorf("esperava email 'joao@email.com', got: %s", out.Email)
		}
	})

	t.Run("retorna erro quando email já está cadastrado", func(t *testing.T) {
		uc := novoUseCase()
		input := ucprovider.CadastrarInput{
			Nome:  "João Silva",
			Email: "joao@email.com",
			Senha: "12345678",
		}
		uc.Executar(input)

		_, err := uc.Executar(input)
		if err != ucprovider.ErrEmailJaCadastrado {
			t.Errorf("esperava ErrEmailJaCadastrado, got: %v", err)
		}
	})

	t.Run("retorna erro quando nome é vazio", func(t *testing.T) {
		uc := novoUseCase()
		_, err := uc.Executar(ucprovider.CadastrarInput{
			Nome:  "",
			Email: "joao@email.com",
			Senha: "12345678",
		})
		if err == nil {
			t.Error("esperava erro para nome vazio")
		}
	})

	t.Run("persiste a senha com hash, nunca em texto puro", func(t *testing.T) {
		repo := repository.NovoProviderMemoria()
		uc := ucprovider.NovoCadastrarUseCase(repo, security.NovoHasherArgon2id())
		uc.Executar(ucprovider.CadastrarInput{
			Nome:  "João Silva",
			Email: "joao@email.com",
			Senha: "12345678",
		})

		p, _ := repo.BuscarPorEmail("joao@email.com")
		if p.SenhaHash == "12345678" {
			t.Error("senha não deveria ser persistida em texto puro")
		}
		if p.SenhaHash == "" {
			t.Error("hash de senha não deveria ser vazio")
		}
	})
}
