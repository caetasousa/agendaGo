// Package admin modela o administrador do sistema — um moderador que lista e
// bane/reativa prestadores e clientes. O admin é semeado no boot a partir de
// variáveis de ambiente; não há cadastro nem auto-registro.
package admin

import (
	"errors"
	"time"
)

// Admin representa um administrador do sistema.
type Admin struct {
	ID           string
	Email        string
	SenhaHash    string
	CriadoEm     time.Time
	AtualizadoEm time.Time
}

var (
	// ErrEmailObrigatorio é retornado quando o email do admin está vazio.
	ErrEmailObrigatorio = errors.New("email é obrigatório")
	// ErrSenhaObrigatoria é retornado quando o hash de senha do admin está vazio.
	ErrSenhaObrigatoria = errors.New("senha é obrigatória")
)

// Novo cria um Admin. Recebe o hash da senha já calculado — o domínio não
// conhece o algoritmo de hash usado.
func Novo(id, email, senhaHash string) (*Admin, error) {
	if email == "" {
		return nil, ErrEmailObrigatorio
	}
	if senhaHash == "" {
		return nil, ErrSenhaObrigatoria
	}

	agora := time.Now()
	return &Admin{
		ID:           id,
		Email:        email,
		SenhaHash:    senhaHash,
		CriadoEm:     agora,
		AtualizadoEm: agora,
	}, nil
}
