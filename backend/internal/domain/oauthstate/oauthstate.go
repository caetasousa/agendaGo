// Package oauthstate representa o state de uso único do fluxo OAuth,
// proteção contra CSRF no callback do provedor de login social.
package oauthstate

import "time"

// State é um valor de uso único emitido antes de redirecionar o usuário ao
// provedor OAuth, verificado de volta no callback. Guarda só o hash — o valor
// puro vai para um cookie de curta duração no navegador. Publico (client ou
// provider) fica gravado aqui, no registro verificado server-side — não é
// lido de volta de um cookie separado, para não depender de um dado que o
// navegador guarda sem vínculo criptográfico com o state.
type State struct {
	StateHash string
	Provedor  string
	Publico   string
	CriadoEm  time.Time
	ExpiraEm  time.Time
}

// Novo cria um state com validade de ttl a partir do momento atual.
func Novo(stateHash, provedor, publico string, ttl time.Duration) *State {
	agora := time.Now()
	return &State{
		StateHash: stateHash,
		Provedor:  provedor,
		Publico:   publico,
		CriadoEm:  agora,
		ExpiraEm:  agora.Add(ttl),
	}
}

// Expirado informa se o state já passou da validade em relação a agora.
func (s *State) Expirado(agora time.Time) bool {
	return agora.After(s.ExpiraEm)
}
