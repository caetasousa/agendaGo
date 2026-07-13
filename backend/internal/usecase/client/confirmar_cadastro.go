package client

import (
	"time"

	"agendago/internal/domain/client"
	"agendago/internal/pkg/token"

	"github.com/google/uuid"
)

// ConfirmarCadastroOutput contém os dados da conta criada/convertida.
type ConfirmarCadastroOutput struct {
	ID    string
	Nome  string
	Email string
}

// ConfirmarCadastroUseCase conclui o cadastro: consome o token e materializa a
// conta. Se o email já era de um convidado, converte o registro existente
// (preservando o ID e os agendamentos); se era inédito, cria uma conta nova.
type ConfirmarCadastroUseCase struct {
	clients   repositorioClient
	providers buscadorProvider
	pendentes repositorioCadastroPendente
}

// NovoConfirmarCadastroUseCase cria uma instância de ConfirmarCadastroUseCase com as dependências injetadas.
func NovoConfirmarCadastroUseCase(clients repositorioClient, providers buscadorProvider, pendentes repositorioCadastroPendente) *ConfirmarCadastroUseCase {
	return &ConfirmarCadastroUseCase{clients: clients, providers: providers, pendentes: pendentes}
}

// Executar consome o token (uso único) e cria a conta. Retorna
// ErrCadastroInvalido para token inexistente, expirado, ou se o email já virou
// conta no intervalo (corrida — ex.: dois links, o primeiro já confirmou).
func (uc *ConfirmarCadastroUseCase) Executar(tokenPuro string) (*ConfirmarCadastroOutput, error) {
	pendente, err := uc.pendentes.Consumir(token.Hash(tokenPuro))
	if err != nil {
		return nil, err
	}
	if pendente == nil || pendente.Expirado(time.Now()) {
		return nil, ErrCadastroInvalido
	}

	// entre o pedido e a confirmação, o email pode ter virado um prestador —
	// o email é único no sistema, então não materializa a conta de cliente
	prestador, err := uc.providers.BuscarPorEmail(pendente.Email)
	if err != nil {
		return nil, err
	}
	if prestador != nil {
		return nil, ErrCadastroInvalido
	}

	existente, err := uc.clients.BuscarPorEmail(pendente.Email)
	if err != nil {
		return nil, err
	}

	if existente != nil {
		// já é conta (alguém confirmou antes, ou o email foi cadastrado por
		// outro caminho): não sobrescreve — resposta genérica
		if existente.TemConta() {
			return nil, ErrCadastroInvalido
		}
		// convidado: converte preservando o ID (e o histórico de agendamentos)
		if err := uc.clients.ConverterEmConta(existente.ID, pendente.SenhaHash, pendente.Telefone); err != nil {
			return nil, err
		}
		uc.pendentes.RemoverExpirados()
		return &ConfirmarCadastroOutput{ID: existente.ID, Nome: existente.Nome, Email: existente.Email}, nil
	}

	// email inédito: cria a conta do zero
	c, err := client.NovoComConta(uuid.NewString(), pendente.Nome, pendente.Email, pendente.SenhaHash)
	if err != nil {
		return nil, err
	}
	c.Telefone = pendente.Telefone
	if err := uc.clients.Salvar(c); err != nil {
		return nil, err
	}

	uc.pendentes.RemoverExpirados()
	return &ConfirmarCadastroOutput{ID: c.ID, Nome: c.Nome, Email: c.Email}, nil
}
