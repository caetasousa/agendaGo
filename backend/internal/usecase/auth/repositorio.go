package auth

import (
	"agendago/internal/domain/admin"
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/session"
)

type repositorioSessao interface {
	Salvar(s *session.Session) error
	BuscarPorTokenHash(hash string) (*session.Session, error)
	Remover(hash string) error
	RemoverExpiradas() error
}

type buscadorProvider interface {
	BuscarPorEmail(email string) (*provider.Provider, error)
	BuscarPorID(id string) (*provider.Provider, error)
}

type buscadorClient interface {
	BuscarPorEmail(email string) (*client.Client, error)
	BuscarPorID(id string) (*client.Client, error)
}

type buscadorAdmin interface {
	BuscarPorEmail(email string) (*admin.Admin, error)
	BuscarPorID(id string) (*admin.Admin, error)
}

// hasherSenha gera e verifica hashes de senha.
type hasherSenha interface {
	Gerar(senha string) (string, error)
	Verificar(senha, hash string) (bool, error)
}
