package client

import (
	"errors"
	"time"
)

// Client representa um cliente do sistema de agendamento. SenhaHash vazio
// indica um cliente convidado, sem conta — ele pode agendar sem se autenticar.
type Client struct {
	ID           string
	Nome         string
	Email        string
	SenhaHash    string
	CriadoEm     time.Time
	AtualizadoEm time.Time
}

var (
	// ErrNomeObrigatorio é retornado quando o nome do cliente está vazio.
	ErrNomeObrigatorio = errors.New("nome é obrigatório")
	// ErrEmailObrigatorio é retornado quando o email do cliente está vazio.
	ErrEmailObrigatorio = errors.New("email é obrigatório")
	// ErrSenhaObrigatoria é retornado quando o hash de senha está vazio ao criar um cliente com conta.
	ErrSenhaObrigatoria = errors.New("senha é obrigatória")
)

// NovoComConta cria um Client com conta. Recebe o hash da senha já calculado —
// o domínio não conhece o algoritmo de hash usado.
// Retorna erro se nome, email ou senhaHash estiverem vazios.
func NovoComConta(id, nome, email, senhaHash string) (*Client, error) {
	if senhaHash == "" {
		return nil, ErrSenhaObrigatoria
	}
	return novo(id, nome, email, senhaHash)
}

// NovoConvidado cria um Client sem conta, para agendar sem se autenticar.
// Retorna erro se nome ou email estiverem vazios.
func NovoConvidado(id, nome, email string) (*Client, error) {
	return novo(id, nome, email, "")
}

func novo(id, nome, email, senhaHash string) (*Client, error) {
	if nome == "" {
		return nil, ErrNomeObrigatorio
	}
	if email == "" {
		return nil, ErrEmailObrigatorio
	}

	agora := time.Now()
	return &Client{
		ID:           id,
		Nome:         nome,
		Email:        email,
		SenhaHash:    senhaHash,
		CriadoEm:     agora,
		AtualizadoEm: agora,
	}, nil
}

// TemConta informa se o cliente possui conta (senha cadastrada).
func (c *Client) TemConta() bool {
	return c.SenhaHash != ""
}
