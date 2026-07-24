// Package socialidentity representa o vínculo entre um usuário (prestador ou
// cliente) e sua identidade num provedor de login social (OIDC).
package socialidentity

import "time"

// Provedor identifica o provedor de identidade OIDC.
type Provedor string

const (
	// Google identifica o provedor Google.
	Google Provedor = "google"
)

// Identidade vincula o sub de um provedor OIDC a um usuário já existente no
// sistema. UserType distingue provider de client, como em session.Session.
type Identidade struct {
	ID       string
	Provedor Provedor
	Sub      string
	UserID   string
	UserType string
	Email    string
	CriadoEm time.Time
}

// Nova cria uma identidade social recém-vinculada.
func Nova(id string, provedor Provedor, sub, userID, userType, email string) *Identidade {
	return &Identidade{
		ID:       id,
		Provedor: provedor,
		Sub:      sub,
		UserID:   userID,
		UserType: userType,
		Email:    email,
		CriadoEm: time.Now(),
	}
}

// IdentidadeOIDC é o resultado de trocar um código de autorização por um
// id_token verificado no provedor — os dados de identidade antes de
// virarem uma Identidade vinculada a um usuário do sistema.
type IdentidadeOIDC struct {
	Sub             string
	Email           string
	EmailVerificado bool
	Nome            string
}
