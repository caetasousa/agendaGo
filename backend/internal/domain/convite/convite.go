// Package convite modela o convite de uso único que dá a uma pessoa acesso à
// agenda de outra. Guarda apenas o hash do token — o token em texto puro
// existe só dentro do email enviado.
package convite

import (
	"errors"
	"time"

	"agendago/internal/domain/membro"
)

var (
	// ErrEmailObrigatorio é retornado quando o convite não tem destinatário.
	ErrEmailObrigatorio = errors.New("email é obrigatório")
	// ErrProviderObrigatorio é retornado quando o convite não aponta para uma agenda.
	ErrProviderObrigatorio = errors.New("prestador é obrigatório")
	// ErrPapelInvalido é retornado quando o papel oferecido não é reconhecido.
	ErrPapelInvalido = errors.New("papel inválido")
	// ErrNaoConvidaDono é retornado ao tentar convidar alguém como dono. Uma
	// agenda tem um dono só, definido no cadastro; transferir a propriedade é
	// outra operação, com outras consequências.
	ErrNaoConvidaDono = errors.New("não é possível convidar alguém como dono da agenda")
)

// Convite liga um token ao email convidado, à agenda de destino e ao papel
// oferecido.
type Convite struct {
	TokenHash  string
	Email      string
	ProviderID string
	Papel      membro.Papel
	CriadoEm   time.Time
	ExpiraEm   time.Time
}

// Novo cria um convite com validade de ttl a partir de agora. Retorna erro se
// faltar email ou agenda, se o papel não for reconhecido, ou se for uma
// tentativa de convidar como dono.
func Novo(tokenHash, email, providerID string, papel membro.Papel, ttl time.Duration) (*Convite, error) {
	if email == "" {
		return nil, ErrEmailObrigatorio
	}
	if providerID == "" {
		return nil, ErrProviderObrigatorio
	}
	if !papel.Valido() {
		return nil, ErrPapelInvalido
	}
	if papel == membro.PapelDono {
		return nil, ErrNaoConvidaDono
	}

	agora := time.Now()
	return &Convite{
		TokenHash:  tokenHash,
		Email:      email,
		ProviderID: providerID,
		Papel:      papel,
		CriadoEm:   agora,
		ExpiraEm:   agora.Add(ttl),
	}, nil
}

// Expirado informa se o convite já passou da validade.
func (c *Convite) Expirado(agora time.Time) bool {
	return agora.After(c.ExpiraEm)
}
