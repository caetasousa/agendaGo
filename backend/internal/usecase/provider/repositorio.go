package provider

import (
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
)

type repositorioCadastrar interface {
	Salvar(p *provider.Provider) error
	BuscarPorEmail(email string) (*provider.Provider, error)
}

// buscadorClient verifica se o email já pertence a um cliente/convidado — o
// email é único entre clientes e prestadores.
type buscadorClient interface {
	BuscarPorEmail(email string) (*client.Client, error)
}

// repositorioPreferencias busca e persiste as preferências mutáveis do prestador.
type repositorioPreferencias interface {
	BuscarPorID(id string) (*provider.Provider, error)
	Atualizar(p *provider.Provider) error
}

// hasherSenha gera o hash da senha em texto puro para persistência.
type hasherSenha interface {
	Gerar(senha string) (string, error)
}
