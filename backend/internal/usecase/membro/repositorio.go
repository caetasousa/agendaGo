// Package membro contém os usecases de equipe: convidar alguém para operar uma
// agenda, aceitar o convite, listar quem tem acesso e remover o acesso.
package membro

import (
	"errors"
	"time"

	"agendago/internal/domain/admin"
	"agendago/internal/domain/client"
	"agendago/internal/domain/convite"
	"agendago/internal/domain/membro"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/usuario"
)

var (
	// ErrEmailIndisponivel é retornado quando o email convidado já tem conta no
	// sistema — como prestador ou como cliente.
	//
	// A mensagem não diz QUAL dos dois, de propósito: quem convida precisa saber
	// que aquele endereço não serve, não descobrir que tipo de conta a pessoa
	// tem. Sem isso, o convite viraria uma sonda de emails cadastrados para
	// qualquer prestador autenticado.
	ErrEmailIndisponivel = errors.New("não foi possível convidar este email; se a pessoa já usa o agendaGo, peça que ela use outro endereço")

	// ErrConviteInvalido é retornado tanto para convite inexistente quanto para
	// expirado — genérico, para não descrever a um estranho o estado de um
	// token que ele possa estar testando.
	ErrConviteInvalido = errors.New("convite inválido ou expirado")

	// ErrProviderNaoEncontrado é retornado quando a agenda do convite sumiu.
	ErrProviderNaoEncontrado = errors.New("agenda não encontrada")

	// ErrMembroNaoEncontrado é retornado ao remover um vínculo que não existe
	// nesta agenda.
	ErrMembroNaoEncontrado = errors.New("membro não encontrado nesta agenda")

	// ErrNaoRemoveDono é retornado ao tentar remover o dono da própria agenda.
	// Uma agenda sem dono não teria quem administrasse a conta.
	ErrNaoRemoveDono = errors.New("o dono da agenda não pode ser removido")
)

// TTLConvite é a validade de um convite a partir da emissão. Generoso porque a
// pessoa convidada não está esperando o email, ao contrário de quem acabou de
// clicar em "cadastrar".
const TTLConvite = 7 * 24 * time.Hour

// repositorioConvite persiste e consome os convites. Consumir apaga ao ler,
// garantindo uso único.
type repositorioConvite interface {
	Salvar(c *convite.Convite) error
	BuscarPorTokenHash(hash string) (*convite.Convite, error)
	Consumir(hash string) (*convite.Convite, error)
	ListarPendentesPorProvider(providerID string) ([]*convite.Convite, error)
	RemoverPorEmail(providerID, email string) error
	RemoverExpirados() error
}

// repositorioUsuario busca e cria contas. O convite consulta antes para
// recusar email já usado, e cria no aceite.
type repositorioUsuario interface {
	BuscarPorEmail(email string) (*usuario.Usuario, error)
	BuscarPorID(id string) (*usuario.Usuario, error)
	Salvar(u *usuario.Usuario) error
}

// repositorioMembro persiste e consulta os vínculos de uma agenda.
type repositorioMembro interface {
	Salvar(m *membro.Membro) error
	BuscarPorUsuario(usuarioID string) (*membro.Membro, error)
	ListarPorProvider(providerID string) ([]*membro.Membro, error)
}

// removedorMembro apaga um vínculo — o acesso revogado.
type removedorMembro interface {
	Remover(id string) error
}

// removedorUsuario apaga a conta de quem ficou sem nenhum vínculo. Ver o
// comentário de RemoverMembroUseCase.Executar para o porquê.
type removedorUsuario interface {
	Remover(id string) error
}

// buscadorProvider resolve a agenda para exibir o nome dela no convite.
type buscadorProvider interface {
	BuscarPorID(id string) (*provider.Provider, error)
}

// buscadorClient verifica se o email já é de um cliente. O email é único entre
// clientes e prestadores, e é essa invariante que faz o login unificado
// funcionar.
type buscadorClient interface {
	BuscarPorEmail(email string) (*client.Client, error)
}

// buscadorAdmin verifica se o email é o do administrador, que é reservado.
type buscadorAdmin interface {
	BuscarPorEmail(email string) (*admin.Admin, error)
}

// enviadorConvite manda o email com o link do convite.
type enviadorConvite interface {
	EnviarConviteMembro(email, nomeAgenda, token string, expiraEm time.Time)
}

// hasherSenha gera o hash da senha escolhida por quem aceita o convite.
type hasherSenha interface {
	Gerar(senha string) (string, error)
}

// revogadorSessoes encerra as sessões de quem perde o acesso — sem isso, o
// membro removido continuaria operando a agenda até a sessão expirar.
type revogadorSessoes interface {
	RemoverDoUsuario(userID string) error
}
