package client

import (
	"agendago/internal/domain/client"

	"github.com/google/uuid"
)

// materializarConta cria a conta com os dados informados, ou converte um
// convidado existente preservando o ID (e o histórico de agendamentos).
// Compartilhado por ConfirmarCadastroUseCase (segunda prova por email) e
// ConcluirPreCadastroUseCase (a posse do email já foi provada pelo token de
// pré-cadastro). Retorna ErrCadastroInvalido se o email já é conta ativa —
// alguém confirmou antes, ou foi cadastrado por outro caminho nesse meio-tempo
// — e também se o convidado está banido: banimento não é revertido por
// cadastro, mesmo que o dono do email tenha provado posse (mesma regra de
// SolicitarCadastroUseCase para o caminho por email).
func materializarConta(clients repositorioClient, nome, email, telefone, senhaHash string) (*client.Client, error) {
	existente, err := clients.BuscarPorEmail(email)
	if err != nil {
		return nil, err
	}

	if existente != nil {
		if existente.TemConta() {
			return nil, ErrCadastroInvalido
		}
		if !existente.Ativo {
			return nil, ErrCadastroInvalido
		}
		if err := clients.ConverterEmConta(existente.ID, senhaHash, telefone); err != nil {
			return nil, err
		}
		return existente, nil
	}

	c, err := client.NovoComConta(uuid.NewString(), nome, email, senhaHash)
	if err != nil {
		return nil, err
	}
	c.Telefone = telefone
	if err := clients.Salvar(c); err != nil {
		return nil, err
	}
	return c, nil
}
