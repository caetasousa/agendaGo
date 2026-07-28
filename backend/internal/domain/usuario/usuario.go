// Identidade de quem loga, separada da agenda que a pessoa opera. Um Usuario
// não sabe nada de horários, duração ou expediente: isso é do Provider, e o
// que liga os dois é o domínio membro.
package usuario

import (
	"errors"
	"time"
)

// Usuario representa quem loga no sistema pelo lado prestador. Ativo distingue
// banimento (moderação pelo admin) da agenda desativada pelo próprio
// prestador: um usuário inativo não loga, e a agenda que ele opera some da
// vitrine — mas as duas decisões vivem em lugares diferentes.
type Usuario struct {
	ID           string
	Email        string
	SenhaHash    string
	Telefone     string
	Ativo        bool
	CriadoEm     time.Time
	AtualizadoEm time.Time
}

var (
	// ErrEmailObrigatorio é retornado quando o email do usuário está vazio.
	ErrEmailObrigatorio = errors.New("email é obrigatório")
	// ErrSenhaObrigatoria é retornado quando o hash de senha do usuário está vazio.
	ErrSenhaObrigatoria = errors.New("senha é obrigatória")
	// ErrTelefoneObrigatorio é retornado quando o telefone está vazio ou é muito curto.
	ErrTelefoneObrigatorio = errors.New("telefone é obrigatório")
)

// Novo cria um Usuario ativo. Recebe o hash da senha já calculado — o domínio
// não conhece o algoritmo de hash usado. Retorna erro se o email ou o
// senhaHash estiverem vazios, ou se o telefone for inválido (validação leve:
// ao menos 8 dígitos).
func Novo(id, email, telefone, senhaHash string) (*Usuario, error) {
	if email == "" {
		return nil, ErrEmailObrigatorio
	}
	if senhaHash == "" {
		return nil, ErrSenhaObrigatoria
	}
	if !telefoneValido(telefone) {
		return nil, ErrTelefoneObrigatorio
	}

	agora := time.Now()
	return &Usuario{
		ID:           id,
		Email:        email,
		SenhaHash:    senhaHash,
		Telefone:     telefone,
		Ativo:        true,
		CriadoEm:     agora,
		AtualizadoEm: agora,
	}, nil
}

// telefoneValido faz uma validação leve: exige ao menos 8 dígitos, ignorando
// formatação. Não verifica se o número existe.
func telefoneValido(telefone string) bool {
	digitos := 0
	for _, r := range telefone {
		if r >= '0' && r <= '9' {
			digitos++
		}
	}
	return digitos >= 8
}

// DefinirTelefone atualiza o telefone de contato do usuário (Preferências).
// Retorna erro se o telefone for inválido.
func (u *Usuario) DefinirTelefone(telefone string) error {
	if !telefoneValido(telefone) {
		return ErrTelefoneObrigatorio
	}
	u.Telefone = telefone
	u.AtualizadoEm = time.Now()
	return nil
}

// DefinirSenha troca o hash da senha do usuário. Recebe o hash já calculado.
// Retorna erro se o hash estiver vazio.
func (u *Usuario) DefinirSenha(senhaHash string) error {
	if senhaHash == "" {
		return ErrSenhaObrigatoria
	}
	u.SenhaHash = senhaHash
	u.AtualizadoEm = time.Now()
	return nil
}

// Banir desativa o usuário (moderação pelo admin): ele deixa de logar, e a
// agenda que opera some da vitrine. Reversível por Reativar.
func (u *Usuario) Banir() {
	u.Ativo = false
	u.AtualizadoEm = time.Now()
}

// Reativar reverte um banimento, devolvendo o acesso do usuário.
func (u *Usuario) Reativar() {
	u.Ativo = true
	u.AtualizadoEm = time.Now()
}
