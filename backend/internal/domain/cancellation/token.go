// Package cancellation modela o token que permite ao convidado cancelar um
// agendamento pelo link no email, sem ter conta. Guarda apenas o hash — o
// token em texto puro nunca é persistido.
package cancellation

import "time"

// Token liga um token de cancelamento ao agendamento correspondente. Não tem
// expiração própria: vale enquanto o agendamento for cancelável, o que é
// decidido pelo domínio do agendamento (regra de antecedência).
type Token struct {
	TokenHash     string
	AppointmentID string
	CriadoEm      time.Time
}

// Novo cria um Token para o agendamento informado, com o momento atual.
func Novo(tokenHash, appointmentID string) *Token {
	return &Token{
		TokenHash:     tokenHash,
		AppointmentID: appointmentID,
		CriadoEm:      time.Now(),
	}
}
