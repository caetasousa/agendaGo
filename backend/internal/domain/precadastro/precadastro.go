// Package precadastro modela o token de uso único que carrega os dados de um
// convidado (nome, email, telefone) do email direto para a tela de cadastro,
// evitando redigitá-los. Guarda apenas o hash do token — o token em texto
// puro nunca é persistido.
package precadastro

import "time"

// PreCadastro liga um token aos dados de contato de um convidado, para
// pré-preencher o formulário de cadastro sem redigitá-los. Não tem
// expiração própria: vale enquanto o token de cancelamento do mesmo email
// valer (mesmo padrão de cancellation.Token) — a proteção é o uso único.
type PreCadastro struct {
	TokenHash string
	Nome      string
	Email     string
	Telefone  string
	CriadoEm  time.Time
}

// Novo cria um PreCadastro para o convidado informado, com o momento atual.
func Novo(tokenHash, nome, email, telefone string) *PreCadastro {
	return &PreCadastro{
		TokenHash: tokenHash,
		Nome:      nome,
		Email:     email,
		Telefone:  telefone,
		CriadoEm:  time.Now(),
	}
}
