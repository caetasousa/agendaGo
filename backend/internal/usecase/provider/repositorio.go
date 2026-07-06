package provider

import "agendago/internal/domain/provider"

type repositorioCadastrar interface {
	Salvar(p *provider.Provider) error
	BuscarPorEmail(email string) (*provider.Provider, error)
}

// hasherSenha gera o hash da senha em texto puro para persistência.
type hasherSenha interface {
	Gerar(senha string) (string, error)
}
