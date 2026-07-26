// Package signup modela um cadastro de cliente ou prestador aguardando
// confirmação por email: os dados da conta a criar (senha já hasheada) ficam
// pendentes até a pessoa provar posse do email clicando no link. Guarda apenas
// o hash do token — o token em texto puro nunca é persistido.
package signup

import (
	"time"

	"agendago/internal/domain/session"
)

// Pendente é um cadastro à espera de confirmação. SenhaHash já vem hasheado —
// a senha em texto puro nunca chega até aqui. Tipo diz qual conta nasce
// quando o token for consumido: cliente ou prestador.
type Pendente struct {
	TokenHash string
	Nome      string
	Email     string
	Telefone  string
	SenhaHash string
	Tipo      session.TipoUsuario
	CriadoEm  time.Time
	ExpiraEm  time.Time
}

// Novo cria um Pendente com validade de ttl a partir do momento atual.
func Novo(tokenHash, nome, email, telefone, senhaHash string, tipo session.TipoUsuario, ttl time.Duration) *Pendente {
	agora := time.Now()
	return &Pendente{
		TokenHash: tokenHash,
		Nome:      nome,
		Email:     email,
		Telefone:  telefone,
		SenhaHash: senhaHash,
		Tipo:      tipo,
		CriadoEm:  agora,
		ExpiraEm:  agora.Add(ttl),
	}
}

// Expirado informa se o cadastro pendente já passou da validade.
func (p *Pendente) Expirado(agora time.Time) bool {
	return agora.After(p.ExpiraEm)
}
