package admin

import (
	"errors"

	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
)

var (
	// ErrProviderNaoEncontrado é retornado quando o prestador alvo não existe.
	ErrProviderNaoEncontrado = errors.New("prestador não encontrado")
	// ErrClientNaoEncontrado é retornado quando o cliente alvo não existe.
	ErrClientNaoEncontrado = errors.New("cliente não encontrado")
)

// repositorioProvider lista, busca e persiste prestadores para a moderação.
type repositorioProvider interface {
	Listar() ([]*provider.Provider, error)
	BuscarPorID(id string) (*provider.Provider, error)
	Atualizar(p *provider.Provider) error
}

// repositorioClient lista, busca e persiste clientes para a moderação.
type repositorioClient interface {
	Listar() ([]*client.Client, error)
	BuscarPorID(id string) (*client.Client, error)
	Atualizar(c *client.Client) error
}

// revogadorSessoes encerra as sessões ativas de um usuário. O banimento revoga
// as sessões na hora — sem isso o banido manteria acesso até o TTL vencer.
type revogadorSessoes interface {
	RemoverDoUsuario(userID string) error
}

// UsuarioResumo descreve um prestador ou cliente na visão de moderação.
type UsuarioResumo struct {
	ID                 string
	Nome               string
	Email              string
	Ativo              bool
	AceitaAgendamentos bool // sempre false para clientes
}

// ModerarUseCase lista e bane/reativa prestadores e clientes.
type ModerarUseCase struct {
	providers repositorioProvider
	clients   repositorioClient
	sessoes   revogadorSessoes
}

// NovoModerarUseCase cria uma instância de ModerarUseCase com as dependências injetadas.
func NovoModerarUseCase(providers repositorioProvider, clients repositorioClient, sessoes revogadorSessoes) *ModerarUseCase {
	return &ModerarUseCase{providers: providers, clients: clients, sessoes: sessoes}
}

// ListarPrestadores devolve todos os prestadores com o status de moderação.
func (uc *ModerarUseCase) ListarPrestadores() ([]UsuarioResumo, error) {
	ps, err := uc.providers.Listar()
	if err != nil {
		return nil, err
	}
	resumos := make([]UsuarioResumo, 0, len(ps))
	for _, p := range ps {
		resumos = append(resumos, UsuarioResumo{
			ID:                 p.ID,
			Nome:               p.Nome,
			Email:              p.Email,
			Ativo:              p.Ativo,
			AceitaAgendamentos: p.AceitaAgendamentos,
		})
	}
	return resumos, nil
}

// ListarClientes devolve todos os clientes com conta e o status de moderação.
func (uc *ModerarUseCase) ListarClientes() ([]UsuarioResumo, error) {
	cs, err := uc.clients.Listar()
	if err != nil {
		return nil, err
	}
	resumos := make([]UsuarioResumo, 0, len(cs))
	for _, c := range cs {
		resumos = append(resumos, UsuarioResumo{
			ID:    c.ID,
			Nome:  c.Nome,
			Email: c.Email,
			Ativo: c.Ativo,
		})
	}
	return resumos, nil
}

// BanirPrestador desativa um prestador e revoga as sessões ativas dele.
// ativo=false remove o acesso e a oferta; reversível por ReativarPrestador.
// Retorna ErrProviderNaoEncontrado se o id não existe.
func (uc *ModerarUseCase) BanirPrestador(id string) error {
	if err := uc.mudarPrestador(id, func(p *provider.Provider) { p.Banir() }); err != nil {
		return err
	}
	return uc.sessoes.RemoverDoUsuario(id)
}

// ReativarPrestador reverte o banimento de um prestador.
func (uc *ModerarUseCase) ReativarPrestador(id string) error {
	return uc.mudarPrestador(id, func(p *provider.Provider) { p.Reativar() })
}

// BanirCliente desativa um cliente (bloqueia o login) e revoga as sessões ativas dele.
func (uc *ModerarUseCase) BanirCliente(id string) error {
	if err := uc.mudarCliente(id, func(c *client.Client) { c.Banir() }); err != nil {
		return err
	}
	return uc.sessoes.RemoverDoUsuario(id)
}

// ReativarCliente reverte o banimento de um cliente.
func (uc *ModerarUseCase) ReativarCliente(id string) error {
	return uc.mudarCliente(id, func(c *client.Client) { c.Reativar() })
}

func (uc *ModerarUseCase) mudarPrestador(id string, muda func(*provider.Provider)) error {
	p, err := uc.providers.BuscarPorID(id)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrProviderNaoEncontrado
	}
	muda(p)
	return uc.providers.Atualizar(p)
}

func (uc *ModerarUseCase) mudarCliente(id string, muda func(*client.Client)) error {
	c, err := uc.clients.BuscarPorID(id)
	if err != nil {
		return err
	}
	if c == nil {
		return ErrClientNaoEncontrado
	}
	muda(c)
	return uc.clients.Atualizar(c)
}
