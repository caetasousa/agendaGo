package provider

import (
	"time"

	"agendago/internal/domain/admin"
	"agendago/internal/domain/client"
	"agendago/internal/domain/membro"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/signup"
	"agendago/internal/domain/usuario"
)

// repositorioCadastrar persiste a agenda de um prestador novo.
type repositorioCadastrar interface {
	Salvar(p *provider.Provider) error
}

// repositorioUsuarioCadastro persiste a identidade e verifica se o email já
// está em uso do lado prestador.
type repositorioUsuarioCadastro interface {
	Salvar(u *usuario.Usuario) error
	BuscarPorEmail(email string) (*usuario.Usuario, error)
}

// repositorioMembroCadastro persiste o vínculo entre identidade e agenda.
type repositorioMembroCadastro interface {
	Salvar(m *membro.Membro) error
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

// repositorioPreferencias busca e persiste as preferências mutáveis da agenda.
type repositorioPreferencias interface {
	BuscarPorID(id string) (*provider.Provider, error)
	BuscarPorSlug(slug string) (*provider.Provider, error)
	Atualizar(p *provider.Provider) error
}

// repositorioUsuario busca e persiste os dados mutáveis da conta — hoje só o
// telefone, que a tela de Preferências edita junto com os da agenda.
type repositorioUsuario interface {
	BuscarPorID(id string) (*usuario.Usuario, error)
	Atualizar(u *usuario.Usuario) error
}

// hasherSenha gera o hash da senha em texto puro para persistência.
type hasherSenha interface {
	Gerar(senha string) (string, error)
}
