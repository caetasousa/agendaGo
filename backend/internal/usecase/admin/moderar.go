package admin

import (
	"log/slog"

	"agendago/internal/domain/auditoria"

	"errors"

	"github.com/google/uuid"

	"agendago/internal/domain/client"
	"agendago/internal/domain/membro"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/usuario"
	"agendago/internal/pkg/paging"
)

var (
	// ErrProviderNaoEncontrado é retornado quando o prestador alvo não existe.
	ErrProviderNaoEncontrado = errors.New("prestador não encontrado")
	// ErrClientNaoEncontrado é retornado quando o cliente alvo não existe.
	ErrClientNaoEncontrado = errors.New("cliente não encontrado")
)

// repositorioProvider lista e busca as agendas para a moderação.
type repositorioProvider interface {
	Listar(pag paging.Pagina) ([]*provider.Provider, int, error)
	BuscarPorID(id string) (*provider.Provider, error)
	Atualizar(p *provider.Provider) error
}

// repositorioUsuario busca e persiste a conta do prestador. O banimento é dela,
// não da agenda: é a conta que deixa de logar.
type repositorioUsuario interface {
	BuscarPorID(id string) (*usuario.Usuario, error)
	Atualizar(u *usuario.Usuario) error
}

// repositorioMembro resolve quem é o dono de cada agenda, para a listagem
// juntar conta e agenda numa linha só.
type repositorioMembro interface {
	DonoDe(providerID string) (*usuario.Usuario, error)
}

// repositorioVinculo faz o caminho inverso: da conta para a agenda que ela
// opera, usado no detalhe de um prestador.
type repositorioVinculo interface {
	BuscarPorUsuario(usuarioID string) (*membro.Membro, error)
}

// repositorioClient lista, busca e persiste clientes para a moderação.
type repositorioClient interface {
	Listar(pag paging.Pagina) ([]*client.Client, int, error)
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
	usuarios  repositorioUsuario
	membros   repositorioMembro
	clients   repositorioClient
	sessoes   revogadorSessoes
	auditoria registradorAuditoria
}

// registradorAuditoria grava a trilha das ações de moderação.
type registradorAuditoria interface {
	Registrar(reg *auditoria.Registro) error
}

// NovoModerarUseCase cria uma instância de ModerarUseCase com as dependências injetadas.
func NovoModerarUseCase(
	providers repositorioProvider,
	usuarios repositorioUsuario,
	membros repositorioMembro,
	clients repositorioClient,
	sessoes revogadorSessoes,
	trilha registradorAuditoria,
) *ModerarUseCase {
	return &ModerarUseCase{providers: providers, usuarios: usuarios, membros: membros, clients: clients, sessoes: sessoes, auditoria: trilha}
}

// ListarPrestadores devolve uma página de prestadores com o status de
// moderação e o total.
func (uc *ModerarUseCase) ListarPrestadores(pag paging.Pagina) ([]UsuarioResumo, int, error) {
	ps, total, err := uc.providers.Listar(pag)
	if err != nil {
		return nil, 0, err
	}
	// Uma consulta por agenda para achar o dono. A página da moderação é
	// pequena (paginada), então o N+1 aqui é barato e evita espalhar um join
	// de leitura pelo repositório de agendas.
	resumos := make([]UsuarioResumo, 0, len(ps))
	for _, p := range ps {
		dono, err := uc.membros.DonoDe(p.ID)
		if err != nil {
			return nil, 0, err
		}
		resumo := UsuarioResumo{
			ID:                 p.ID,
			Nome:               p.Nome,
			AceitaAgendamentos: p.AceitaAgendamentos,
		}
		// Agenda sem dono não deveria existir; se existir, aparece na
		// moderação sem email e como inativa, em vez de sumir da lista.
		if dono != nil {
			resumo.ID = dono.ID
			resumo.Email = dono.Email
			resumo.Ativo = dono.Ativo
		}
		resumos = append(resumos, resumo)
	}
	return resumos, total, nil
}

// ListarClientes devolve uma página de clientes com conta e o status de
// moderação, e o total.
func (uc *ModerarUseCase) ListarClientes(pag paging.Pagina) ([]UsuarioResumo, int, error) {
	cs, total, err := uc.clients.Listar(pag)
	if err != nil {
		return nil, 0, err
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
	return resumos, total, nil
}

// BanirPrestador desativa um prestador e revoga as sessões ativas dele.
// ativo=false remove o acesso e a oferta; reversível por ReativarPrestador.
// Retorna ErrProviderNaoEncontrado se o id não existe.
func (uc *ModerarUseCase) BanirPrestador(adminID, id string) error {
	if err := uc.mudarPrestador(id, func(u *usuario.Usuario) { u.Banir() }); err != nil {
		return err
	}
	uc.registrar(adminID, auditoria.AcaoBanirPrestador, auditoria.TipoUsuario, id)
	return uc.sessoes.RemoverDoUsuario(id)
}

// ReativarPrestador reverte o banimento de um prestador.
func (uc *ModerarUseCase) ReativarPrestador(adminID, id string) error {
	if err := uc.mudarPrestador(id, func(u *usuario.Usuario) { u.Reativar() }); err != nil {
		return err
	}
	uc.registrar(adminID, auditoria.AcaoReativarPrestador, auditoria.TipoUsuario, id)
	return nil
}

// BanirCliente desativa um cliente (bloqueia o login) e revoga as sessões ativas dele.
func (uc *ModerarUseCase) BanirCliente(adminID, id string) error {
	if err := uc.mudarCliente(id, func(c *client.Client) { c.Banir() }); err != nil {
		return err
	}
	uc.registrar(adminID, auditoria.AcaoBanirCliente, auditoria.TipoCliente, id)
	return uc.sessoes.RemoverDoUsuario(id)
}

// ReativarCliente reverte o banimento de um cliente.
func (uc *ModerarUseCase) ReativarCliente(adminID, id string) error {
	if err := uc.mudarCliente(id, func(c *client.Client) { c.Reativar() }); err != nil {
		return err
	}
	uc.registrar(adminID, auditoria.AcaoReativarCliente, auditoria.TipoCliente, id)
	return nil
}

// registrar grava a moderação na trilha. É interceptado no USECASE, e não no
// handler, porque é aqui que a intenção tem nome: o handler só sabe que veio um
// POST numa rota.
//
// Falha na trilha não desfaz a moderação — o banimento já valeu, e devolver
// erro sugeriria o contrário. Vai para o log, que é onde se descobre trilha
// faltando.
func (uc *ModerarUseCase) registrar(adminID string, acao auditoria.Acao, alvoTipo, alvoID string) {
	if uc.auditoria == nil {
		return
	}
	reg, err := auditoria.Novo(uuid.NewString(), adminID, auditoria.TipoAdmin, acao, alvoTipo, alvoID, nil)
	if err != nil {
		slog.Error("falha ao montar registro de auditoria", slog.String("erro", err.Error()))
		return
	}
	if err := uc.auditoria.Registrar(reg); err != nil {
		slog.Error("falha ao gravar auditoria da moderação",
			slog.String("erro", err.Error()), slog.String("acao", string(acao)))
	}
}

// mudarPrestador aplica a moderação na CONTA. O id que chega é o da conta —
// para os prestadores anteriores à separação ele coincide com o da agenda,
// porque a migração reusou o mesmo UUID.
func (uc *ModerarUseCase) mudarPrestador(id string, muda func(*usuario.Usuario)) error {
	u, err := uc.usuarios.BuscarPorID(id)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrProviderNaoEncontrado
	}
	muda(u)
	return uc.usuarios.Atualizar(u)
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
