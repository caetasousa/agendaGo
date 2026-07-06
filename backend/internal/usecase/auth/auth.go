// Package auth contém os usecases de autenticação: login, logout, validação
// de sessão e consulta do perfil autenticado.
package auth

import (
	"errors"
	"time"

	"agendago/internal/domain/session"
)

// ErrCredenciaisInvalidas é retornado tanto para email inexistente quanto para
// senha incorreta — resposta genérica de propósito, para não revelar quais
// emails estão cadastrados.
var ErrCredenciaisInvalidas = errors.New("credenciais inválidas")

// ErrSessaoInvalida é retornado quando o token não corresponde a nenhuma
// sessão ativa ou a sessão já expirou.
var ErrSessaoInvalida = errors.New("sessão inválida")

// TTLSessao é a validade fixa de uma sessão a partir do login.
const TTLSessao = 7 * 24 * time.Hour

// hashDummy é comparado contra a senha informada quando o email não é
// encontrado, para equalizar o tempo de resposta e evitar que a diferença de
// timing entre "email existe" e "email não existe" seja usada para enumerar
// emails cadastrados.
const hashDummy = "$argon2id$v=19$m=19456,t=2,p=1$418kZE16sLJFdfFuo0wHMQ$GStCwTn2VCAjSSyD9oE28ZkZ2ron++CtYmNlS62v0LY"

// LoginInput contém as credenciais informadas no login.
type LoginInput struct {
	Email string
	Senha string
}

// LoginOutput contém o resultado de um login bem-sucedido. Token é o token de
// sessão em texto puro — vai apenas para o cookie de resposta, nunca é persistido.
type LoginOutput struct {
	Token    string
	ExpiraEm time.Time
	UserID   string
	Nome     string
}

// Identidade representa o usuário autenticado em uma requisição.
type Identidade struct {
	UserID string
	Tipo   session.TipoUsuario
}
