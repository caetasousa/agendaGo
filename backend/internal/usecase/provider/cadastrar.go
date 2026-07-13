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
	Nome     string
	Email    string
	Telefone string
	Senha    string
}

// CadastrarOutput contém os dados do prestador após o cadastro.
type CadastrarOutput struct {
	ID    string
	Nome  string
	Email string
}

// CadastrarUseCase orquestra o cadastro de um novo prestador.
type CadastrarUseCase struct {
	repo    repositorioCadastrar
	clients buscadorClient
	hasher  hasherSenha
}

// NovoCadastrarUseCase cria uma instância de CadastrarUseCase com o repositório, o buscador de clientes e o hasher de senha injetados.
func NovoCadastrarUseCase(repo repositorioCadastrar, clients buscadorClient, hasher hasherSenha) *CadastrarUseCase {
	return &CadastrarUseCase{repo: repo, clients: clients, hasher: hasher}
}

// Executar valida os dados, verifica duplicidade de email (entre prestadores e
// clientes — o email é único no sistema), hasheia a senha e persiste o novo
// prestador. Retorna ErrEmailJaCadastrado se o email já estiver em uso.
func (uc *CadastrarUseCase) Executar(input CadastrarInput) (*CadastrarOutput, error) {
	existente, err := uc.repo.BuscarPorEmail(input.Email)
	if err != nil {
		return nil, err
	}
	if existente != nil {
		return nil, ErrEmailJaCadastrado
	}

	// o email não pode já pertencer a um cliente/convidado
	clienteExistente, err := uc.clients.BuscarPorEmail(input.Email)
	if err != nil {
		return nil, err
	}
	if clienteExistente != nil {
		return nil, ErrEmailJaCadastrado
	}

	senhaHash, err := uc.hasher.Gerar(input.Senha)
	if err != nil {
		return nil, err
	}

	p, err := provider.Novo(uuid.NewString(), input.Nome, input.Email, input.Telefone, senhaHash)
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
