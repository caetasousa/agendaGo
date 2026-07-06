package client

import "agendago/internal/domain/client"

type repositorioCadastrar interface {
	Salvar(c *client.Client) error
	BuscarPorEmail(email string) (*client.Client, error)
}

// hasherSenha gera o hash da senha em texto puro para persistência.
type hasherSenha interface {
	Gerar(senha string) (string, error)
}
