// Package cancellation modela o token que permite ao convidado cancelar um
// agendamento pelo link no email, sem ter conta. Guarda apenas o hash — o
// token em texto puro nunca é persistido.
package cancellation

import "time"

// Token liga um token de cancelamento ao agendamento correspondente.
type Token struct {
	TokenHash     string
	AppointmentID string
	CriadoEm      time.Time
	ExpiraEm      time.Time
}

// Novo cria um Token para o agendamento informado, com validade de ttl a
// partir do momento atual.
func Novo(tokenHash, appointmentID string, ttl time.Duration) *Token {
	agora := time.Now()
	return &Token{
		TokenHash:     tokenHash,
		AppointmentID: appointmentID,
		CriadoEm:      agora,
		ExpiraEm:      agora.Add(ttl),
	}
}

// Expirado informa se o token já passou da validade.
func (t *Token) Expirado(agora time.Time) bool {
	return agora.After(t.ExpiraEm)
}
