package provider

import (
	"time"

	"agendago/internal/domain/admin"
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/signup"
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

// buscadorAdmin verifica se o email pertence ao administrador. O email do admin
// é reservado: nenhum cadastro pode criar uma conta de prestador com ele.
type buscadorAdmin interface {
	BuscarPorEmail(email string) (*admin.Admin, error)
}

// repositorioCadastroPendente persiste e consome os cadastros de prestador à
// espera de confirmação por email. Consumir apaga o registro ao lê-lo (uso único).
type repositorioCadastroPendente interface {
	Salvar(p *signup.Pendente) error
	Consumir(tokenHash string) (*signup.Pendente, error)
	RemoverPorEmail(email string) error
	RemoverExpirados() error
}

// enviadorCadastro envia os emails do fluxo de cadastro de prestador.
type enviadorCadastro interface {
	// EnviarConfirmacaoCadastroPrestador manda o link de confirmação.
	EnviarConfirmacaoCadastroPrestador(email, nome, token string, expiraEm time.Time)
	// EnviarAvisoContaExistente avisa que o email já tem conta, sem revelar
	// isso na resposta HTTP.
	EnviarAvisoContaExistente(email, nome string)
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
