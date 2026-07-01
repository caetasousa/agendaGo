package provider

import "agendago/internal/domain/provider"

type repositorioCadastrar interface {
	Salvar(p *provider.Provider) error
	BuscarPorEmail(email string) (*provider.Provider, error)
}
