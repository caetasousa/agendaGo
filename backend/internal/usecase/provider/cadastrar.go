package provider

import (
	"errors"

	"agendago/internal/domain/provider"

	"github.com/google/uuid"
)

// ErrEmailJaCadastrado é retornado quando o email informado já está em uso.
var ErrEmailJaCadastrado = errors.New("email já cadastrado")

// CadastrarInput contém os dados necessários para cadastrar um prestador.
type CadastrarInput struct {
	Nome  string
	Email string
	Senha string
}

// CadastrarOutput contém os dados do prestador após o cadastro.
type CadastrarOutput struct {
	ID    string
	Nome  string
	Email string
}

// CadastrarUseCase orquestra o cadastro de um novo prestador.
type CadastrarUseCase struct {
	repo   repositorioCadastrar
	hasher hasherSenha
}

// NovoCadastrarUseCase cria uma instância de CadastrarUseCase com o repositório e o hasher de senha injetados.
func NovoCadastrarUseCase(repo repositorioCadastrar, hasher hasherSenha) *CadastrarUseCase {
	return &CadastrarUseCase{repo: repo, hasher: hasher}
}

// Executar valida os dados, verifica duplicidade de email, hasheia a senha e persiste o novo prestador.
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

	p, err := provider.Novo(uuid.NewString(), input.Nome, input.Email, senhaHash)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Salvar(p); err != nil {
		return nil, err
	}

	return &CadastrarOutput{
		ID:    p.ID,
		Nome:  p.Nome,
		Email: p.Email,
	}, nil
}
