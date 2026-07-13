package client

import (
	"errors"
	"time"
)

// Client representa um cliente do sistema de agendamento. SenhaHash vazio
// indica um cliente convidado, sem conta — ele pode agendar sem se autenticar.
// Ativo=false é um banimento pelo admin: o cliente deixa de logar. Telefone é
// opcional para quem tem conta, mas exigido no agendamento de convidado, para
// o prestador ter como contatar.
type Client struct {
	ID           string
	Nome         string
	Email        string
	Telefone     string
	SenhaHash    string
	Ativo        bool
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
	// ErrTelefoneObrigatorio é retornado quando o telefone do convidado está vazio ou é muito curto.
	ErrTelefoneObrigatorio = errors.New("telefone é obrigatório")
)

// NovoComConta cria um Client com conta. Recebe o hash da senha já calculado —
// o domínio não conhece o algoritmo de hash usado.
// Retorna erro se nome, email ou senhaHash estiverem vazios.
func NovoComConta(id, nome, email, senhaHash string) (*Client, error) {
	if senhaHash == "" {
		return nil, ErrSenhaObrigatoria
	}
	return novo(id, nome, email, "", senhaHash)
}

// NovoConvidado cria um Client sem conta, para agendar sem se autenticar. O
// telefone é obrigatório (validação leve: ao menos 8 dígitos) — é como o
// prestador entra em contato com quem agendou sem cadastro.
func NovoConvidado(id, nome, email, telefone string) (*Client, error) {
	if !telefoneValido(telefone) {
		return nil, ErrTelefoneObrigatorio
	}
	return novo(id, nome, email, telefone, "")
}

// telefoneValido faz uma validação leve: exige ao menos 8 dígitos, ignorando
// formatação (espaços, parênteses, traços). Não verifica se o número existe.
func telefoneValido(telefone string) bool {
	digitos := 0
	for _, r := range telefone {
		if r >= '0' && r <= '9' {
			digitos++
		}
	}
	return digitos >= 8
}

func novo(id, nome, email, telefone, senhaHash string) (*Client, error) {
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
		Telefone:     telefone,
		SenhaHash:    senhaHash,
		Ativo:        true,
		CriadoEm:     agora,
		AtualizadoEm: agora,
	}, nil
}

// TemConta informa se o cliente possui conta (senha cadastrada).
func (c *Client) TemConta() bool {
	return c.SenhaHash != ""
}

// DefinirConta converte um convidado em conta, definindo a senha (hash já
// calculado) e o telefone. Usado quando alguém confirma o cadastro com um
// email que já usou como convidado — o histórico de agendamentos é preservado
// porque o mesmo Client (mesmo ID) passa a ter conta. Retorna erro se a senha
// estiver vazia ou o telefone for inválido.
func (c *Client) DefinirConta(senhaHash, telefone string) error {
	if senhaHash == "" {
		return ErrSenhaObrigatoria
	}
	if !telefoneValido(telefone) {
		return ErrTelefoneObrigatorio
	}
	c.SenhaHash = senhaHash
	c.Telefone = telefone
	c.AtualizadoEm = time.Now()
	return nil
}

// Banir desativa o cliente (moderação pelo admin): ele deixa de logar. Reversível por Reativar.
func (c *Client) Banir() {
	c.Ativo = false
	c.AtualizadoEm = time.Now()
}

// Reativar reverte um banimento, devolvendo o acesso do cliente.
func (c *Client) Reativar() {
	c.Ativo = true
	c.AtualizadoEm = time.Now()
}
