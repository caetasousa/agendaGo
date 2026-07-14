// Package precadastro modela o token de uso único que carrega os dados de um
// convidado (nome, email, telefone) do email direto para a tela de cadastro,
// evitando redigitá-los. Guarda apenas o hash do token — o token em texto
// puro nunca é persistido.
package precadastro

import "time"

// PreCadastro liga um token aos dados de contato de um convidado, para
// pré-preencher o formulário de cadastro sem redigitá-los.
type PreCadastro struct {
	TokenHash string
	Nome      string
	Email     string
	Telefone  string
	CriadoEm  time.Time
	ExpiraEm  time.Time
}

// Novo cria um PreCadastro para o convidado informado, com validade de ttl a
// partir do momento atual.
func Novo(tokenHash, nome, email, telefone string, ttl time.Duration) *PreCadastro {
	agora := time.Now()
	return &PreCadastro{
		TokenHash: tokenHash,
		Nome:      nome,
		Email:     email,
		Telefone:  telefone,
		CriadoEm:  agora,
		ExpiraEm:  agora.Add(ttl),
	}
}

// Expirado informa se o token já passou da validade.
func (p *PreCadastro) Expirado(agora time.Time) bool {
	return agora.After(p.ExpiraEm)
}
