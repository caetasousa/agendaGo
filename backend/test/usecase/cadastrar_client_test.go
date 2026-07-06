package usecase_test

import (
	"testing"

	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	ucclient "agendago/internal/usecase/client"
)

func novoClientUseCase() (*ucclient.CadastrarUseCase, *repository.ClientMemoria) {
	repo := repository.NovoClientMemoria()
	return ucclient.NovoCadastrarUseCase(repo, security.NovoHasherArgon2id()), repo
}

func TestCadastrarClient(t *testing.T) {
	t.Run("cadastra client com dados válidos e retorna ID gerado", func(t *testing.T) {
		uc, _ := novoClientUseCase()
		out, err := uc.Executar(ucclient.CadastrarInput{
			Nome:  "Maria Silva",
			Email: "maria@email.com",
			Senha: "12345678",
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.ID == "" {
			t.Error("ID não deve ser vazio")
		}
		if out.Email != "maria@email.com" {
			t.Errorf("esperava email 'maria@email.com', got: %s", out.Email)
		}
	})

	t.Run("retorna erro quando email já está cadastrado", func(t *testing.T) {
		uc, _ := novoClientUseCase()
		input := ucclient.CadastrarInput{
			Nome:  "Maria Silva",
			Email: "maria@email.com",
			Senha: "12345678",
		}
		uc.Executar(input)

		_, err := uc.Executar(input)
		if err != ucclient.ErrEmailJaCadastrado {
			t.Errorf("esperava ErrEmailJaCadastrado, got: %v", err)
		}
	})

	t.Run("retorna erro quando nome é vazio", func(t *testing.T) {
		uc, _ := novoClientUseCase()
		_, err := uc.Executar(ucclient.CadastrarInput{
			Nome:  "",
			Email: "maria@email.com",
			Senha: "12345678",
		})
		if err == nil {
			t.Error("esperava erro para nome vazio")
		}
	})

	t.Run("persiste a senha com hash, nunca em texto puro", func(t *testing.T) {
		uc, repo := novoClientUseCase()
		uc.Executar(ucclient.CadastrarInput{
			Nome:  "Maria Silva",
			Email: "maria@email.com",
			Senha: "12345678",
		})

		c, _ := repo.BuscarPorEmail("maria@email.com")
		if c.SenhaHash == "12345678" {
			t.Error("senha não deveria ser persistida em texto puro")
		}
	})
}
