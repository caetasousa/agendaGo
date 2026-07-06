package provider

import (
	"errors"
	"time"
)

// Provider representa um prestador de serviço no sistema de agendamento.
type Provider struct {
	ID                 string
	Nome               string
	Email              string
	SenhaHash          string
	AceitaAgendamentos bool
	DescansoMinutos    int
	CriadoEm           time.Time
	AtualizadoEm       time.Time
}

var (
	// ErrNomeObrigatorio é retornado quando o nome do prestador está vazio.
	ErrNomeObrigatorio = errors.New("nome é obrigatório")
	// ErrEmailObrigatorio é retornado quando o email do prestador está vazio.
	ErrEmailObrigatorio = errors.New("email é obrigatório")
	// ErrSenhaObrigatoria é retornado quando o hash de senha do prestador está vazio.
	ErrSenhaObrigatoria = errors.New("senha é obrigatória")
	// ErrDescansoInvalido é retornado quando o tempo de descanso é negativo.
	ErrDescansoInvalido = errors.New("descanso não pode ser negativo")
)

// Novo cria um Provider com agenda desativada por padrão. Recebe o hash da
// senha já calculado — o domínio não conhece o algoritmo de hash usado.
// Retorna erro se nome, email ou senhaHash estiverem vazios.
func Novo(id, nome, email, senhaHash string) (*Provider, error) {
	if nome == "" {
		return nil, ErrNomeObrigatorio
	}
	if email == "" {
		return nil, ErrEmailObrigatorio
	}
	if senhaHash == "" {
		return nil, ErrSenhaObrigatoria
	}

	agora := time.Now()
	return &Provider{
		ID:                 id,
		Nome:               nome,
		Email:              email,
		SenhaHash:          senhaHash,
		AceitaAgendamentos: false,
		DescansoMinutos:    0,
		CriadoEm:           agora,
		AtualizadoEm:       agora,
	}, nil
}

// AtivarAgenda habilita o prestador a receber agendamentos.
func (p *Provider) AtivarAgenda() {
	p.AceitaAgendamentos = true
	p.AtualizadoEm = time.Now()
}

// DesativarAgenda impede o prestador de receber novos agendamentos.
func (p *Provider) DesativarAgenda() {
	p.AceitaAgendamentos = false
	p.AtualizadoEm = time.Now()
}

// DefinirDescanso define o intervalo em minutos entre atendimentos.
// Retorna erro se o valor for negativo.
func (p *Provider) DefinirDescanso(minutos int) error {
	if minutos < 0 {
		return ErrDescansoInvalido
	}
	p.DescansoMinutos = minutos
	p.AtualizadoEm = time.Now()
	return nil
}
