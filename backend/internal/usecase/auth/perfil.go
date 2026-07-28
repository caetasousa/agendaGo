package auth

import (
	"agendago/internal/domain/availability"
	"agendago/internal/domain/session"
)

// AgendaDoPerfil descreve a agenda que o usuário autenticado opera e o papel
// dele nela. Só existe para o tipo provider — é o bloco que separa, na
// resposta, o que é da conta do que é da agenda.
type AgendaDoPerfil struct {
	ID                           string
	Papel                        string
	AceitaAgendamentos           bool
	DescansoMinutos              int
	DuracaoAtendimentoMinutos    int
	HorariosPadrao               []availability.TimeBlock
	PermiteMarcacaoPeloPrestador bool
}

// PerfilOutput contém os dados do usuário autenticado. Provider só é
// preenchido para prestadores — fica nil para clientes e admins.
type PerfilOutput struct {
	ID       string
	Nome     string
	Email    string
	Telefone string
	Tipo     string
	Provider *AgendaDoPerfil
	// TelefonePendente é true quando o prestador entrou via login social e
	// ainda não confirmou um telefone de verdade (ver TelefonePendente em
	// login_social.go) — o frontend usa isso para travar o painel em
	// Preferências até ele completar o cadastro.
	TelefonePendente bool
}

// PerfilUseCase consulta os dados do usuário autenticado a partir da sua identidade de sessão.
type PerfilUseCase struct {
	usuarios  buscadorUsuario
	providers buscadorProvider
	clients   buscadorClient
	admins    buscadorAdmin
}

// NovoPerfilUseCase cria uma instância de PerfilUseCase com os buscadores injetados.
func NovoPerfilUseCase(usuarios buscadorUsuario, providers buscadorProvider, clients buscadorClient, admins buscadorAdmin) *PerfilUseCase {
	return &PerfilUseCase{usuarios: usuarios, providers: providers, clients: clients, admins: admins}
}

// Executar busca os dados correspondentes à identidade, conforme o tipo de
// usuário. Para prestadores junta a conta (usuarios) com a agenda que ela
// opera, resolvida pelo vínculo na validação da sessão.
func (uc *PerfilUseCase) Executar(id Identidade) (*PerfilOutput, error) {
	switch id.Tipo {
	case session.TipoProvider:
		u, err := uc.usuarios.BuscarPorID(id.UserID)
		if err != nil {
			return nil, err
		}
		if u == nil {
			return nil, ErrSessaoInvalida
		}
		p, err := uc.providers.BuscarPorID(id.ProviderID)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, ErrSessaoInvalida
		}
		return &PerfilOutput{
			ID:       u.ID,
			Nome:     p.Nome,
			Email:    u.Email,
			Telefone: u.Telefone,
			Tipo:     string(session.TipoProvider),
			Provider: &AgendaDoPerfil{
				ID:                           p.ID,
				Papel:                        string(id.Papel),
				AceitaAgendamentos:           p.AceitaAgendamentos,
				DescansoMinutos:              p.DescansoMinutos,
				DuracaoAtendimentoMinutos:    p.DuracaoAtendimentoMinutos,
				HorariosPadrao:               p.HorariosPadrao,
				PermiteMarcacaoPeloPrestador: p.PermiteMarcacaoPeloPrestador,
			},
			TelefonePendente: u.Telefone == TelefonePendente,
		}, nil

	case session.TipoClient:
		c, err := uc.clients.BuscarPorID(id.UserID)
		if err != nil {
			return nil, err
		}
		if c == nil {
			return nil, ErrSessaoInvalida
		}
		return &PerfilOutput{ID: c.ID, Nome: c.Nome, Email: c.Email, Telefone: c.Telefone, Tipo: string(session.TipoClient)}, nil

	case session.TipoAdmin:
		a, err := uc.admins.BuscarPorID(id.UserID)
		if err != nil {
			return nil, err
		}
		if a == nil {
			return nil, ErrSessaoInvalida
		}
		return &PerfilOutput{ID: a.ID, Nome: "Admin", Email: a.Email, Tipo: string(session.TipoAdmin)}, nil

	default:
		return nil, ErrSessaoInvalida
	}
}
