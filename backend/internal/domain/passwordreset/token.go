// Package passwordreset modela o token de recuperação de senha: opaco, de
// uso único e com validade curta. Espelha internal/domain/session.
package passwordreset

import (
	"time"

	"agendago/internal/domain/session"
)

// Token representa um pedido de recuperação de senha pendente. Guarda apenas
// o hash do token — o token em texto puro nunca é persistido.
type Token struct {
	TokenHash string
	UserID    string
	UserType  session.TipoUsuario
	CriadoEm  time.Time
	ExpiraEm  time.Time
}

// Novo cria um Token com validade de ttl a partir do momento atual.
func Novo(tokenHash, userID string, tipo session.TipoUsuario, ttl time.Duration) *Token {
	agora := time.Now()
	return &Token{
		TokenHash: tokenHash,
		UserID:    userID,
		UserType:  tipo,
		CriadoEm:  agora,
		ExpiraEm:  agora.Add(ttl),
	}
}

// Expirado informa se o token já passou da validade em relação a agora.
func (t *Token) Expirado(agora time.Time) bool {
	return agora.After(t.ExpiraEm)
}
