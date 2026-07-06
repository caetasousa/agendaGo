package client

import (
	"errors"

	"agendago/internal/domain/client"

	"github.com/google/uuid"
)

// ErrEmailJaCadastrado é retornado quando o email informado já está em uso.
var ErrEmailJaCadastrado = errors.New("email já cadastrado")

// CadastrarInput contém os dados necessários para cadastrar um cliente com conta.
type CadastrarInput struct {
	Nome  string
	Email string
	Senha string
}

// CadastrarOutput contém os dados do cliente após o cadastro.
type CadastrarOutput struct {
	ID    string
	Nome  string
	Email string
}

// CadastrarUseCase orquestra o cadastro de um novo cliente com conta.
type CadastrarUseCase struct {
	repo   repositorioCadastrar
	hasher hasherSenha
}

// NovoCadastrarUseCase cria uma instância de CadastrarUseCase com o repositório e o hasher de senha injetados.
func NovoCadastrarUseCase(repo repositorioCadastrar, hasher hasherSenha) *CadastrarUseCase {
	return &CadastrarUseCase{repo: repo, hasher: hasher}
}

// Executar valida os dados, verifica duplicidade de email, hasheia a senha e persiste o novo cliente.
// Retorna erro se o email já estiver cadastrado ou se os dados forem inválidos.
func (uc *CadastrarUseCase) Executar(input CadastrarInput) (*CadastrarOutput, error) {
	existente, err := uc.repo.BuscarPorEmail(input.Email)
	if err != nil {
		return nil, err
	}
	if existente != nil {
		return nil, ErrEmailJaCadastrado
	}

	senhaHash, err := uc.hasher.Gerar(input.Senha)
	if err != nil {
		return nil, err
	}

	c, err := client.NovoComConta(uuid.NewString(), input.Nome, input.Email, senhaHash)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Salvar(c); err != nil {
		return nil, err
	}

	return &CadastrarOutput{
		ID:    c.ID,
		Nome:  c.Nome,
		Email: c.Email,
	}, nil
}
