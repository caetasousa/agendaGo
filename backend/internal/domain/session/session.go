package session

import "time"

// TipoUsuario identifica o perfil dono da sessão.
type TipoUsuario string

const (
	// TipoProvider identifica uma sessão de prestador.
	TipoProvider TipoUsuario = "provider"
	// TipoClient identifica uma sessão de cliente.
	TipoClient TipoUsuario = "client"
	// TipoAdmin identifica uma sessão de administrador (moderação).
	TipoAdmin TipoUsuario = "admin"
)

// Session representa uma sessão autenticada. Guarda apenas o hash do token —
// o token em texto puro nunca é persistido.
type Session struct {
	TokenHash string
	UserID    string
	UserType  TipoUsuario
	CriadoEm  time.Time
	ExpiraEm  time.Time
}

// Nova cria uma sessão com validade de ttl a partir do momento atual.
func Nova(tokenHash, userID string, tipo TipoUsuario, ttl time.Duration) *Session {
	agora := time.Now()
	return &Session{
		TokenHash: tokenHash,
		UserID:    userID,
		UserType:  tipo,
		CriadoEm:  agora,
		ExpiraEm:  agora.Add(ttl),
	}
}

// Expirada informa se a sessão já passou da validade em relação a agora.
func (s *Session) Expirada(agora time.Time) bool {
	return agora.After(s.ExpiraEm)
}
