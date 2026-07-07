package auth

import "agendago/internal/domain/session"

// PerfilOutput contém os dados do usuário autenticado. AceitaAgendamentos e
// DescansoMinutos só são preenchidos para prestadores — ficam nil para clientes.
type PerfilOutput struct {
	ID                 string
	Nome               string
	Email              string
	Tipo               string
	AceitaAgendamentos *bool
	DescansoMinutos    *int
}

// PerfilUseCase consulta os dados do usuário autenticado a partir da sua identidade de sessão.
type PerfilUseCase struct {
	providers buscadorProvider
	clients   buscadorClient
}

// NovoPerfilUseCase cria uma instância de PerfilUseCase com os buscadores de prestador e cliente injetados.
func NovoPerfilUseCase(providers buscadorProvider, clients buscadorClient) *PerfilUseCase {
	return &PerfilUseCase{providers: providers, clients: clients}
}

// Executar busca o prestador ou cliente correspondente à identidade, conforme o tipo de usuário.
func (uc *PerfilUseCase) Executar(id Identidade) (*PerfilOutput, error) {
	switch id.Tipo {
	case session.TipoProvider:
		p, err := uc.providers.BuscarPorID(id.UserID)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, ErrSessaoInvalida
		}
		return &PerfilOutput{
			ID:                 p.ID,
			Nome:               p.Nome,
			Email:              p.Email,
			Tipo:               string(session.TipoProvider),
			AceitaAgendamentos: &p.AceitaAgendamentos,
			DescansoMinutos:    &p.DescansoMinutos,
		}, nil

	case session.TipoClient:
		c, err := uc.clients.BuscarPorID(id.UserID)
		if err != nil {
			return nil, err
		}
		if c == nil {
			return nil, ErrSessaoInvalida
		}
		return &PerfilOutput{ID: c.ID, Nome: c.Nome, Email: c.Email, Tipo: string(session.TipoClient)}, nil

	default:
		return nil, ErrSessaoInvalida
	}
}
