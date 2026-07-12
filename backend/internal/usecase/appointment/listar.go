package appointment

import (
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
)

// Resumo é um agendamento pronto para exibição. Além dos nomes das duas
// pontas, carrega o contato do cliente (email/telefone) — o prestador precisa
// dele para falar com quem agendou, sobretudo convidados sem cadastro.
type Resumo struct {
	ID              string
	Data            time.Time
	InicioMinutos   int
	FimMinutos      int
	Status          appointment.Status
	ExpiraEm        time.Time
	NomeCliente     string
	EmailCliente    string
	TelefoneCliente string
	NomePrestador   string
}

// ListarInput identifica o dono da listagem e o instante da consulta.
type ListarInput struct {
	UsuarioID string
	Agora     time.Time
}

// ListarOutput contém os agendamentos do usuário, ordenados por data e início.
type ListarOutput struct {
	Agendamentos []Resumo
}

// ListarUseCase lista os agendamentos de um prestador ou de um cliente,
// efetivando a expiração lazy: solicitações com TTL vencido viram EXPIRADO
// no ato da leitura.
type ListarUseCase struct {
	repo         repositorioAppointment
	providerRepo repositorioProvider
	clientRepo   repositorioClient
}

// NovoListarUseCase cria uma instância de ListarUseCase com os repositórios injetados.
func NovoListarUseCase(repo repositorioAppointment, providerRepo repositorioProvider, clientRepo repositorioClient) *ListarUseCase {
	return &ListarUseCase{repo: repo, providerRepo: providerRepo, clientRepo: clientRepo}
}

// DoPrestador lista os agendamentos recebidos pelo prestador.
func (uc *ListarUseCase) DoPrestador(in ListarInput) (*ListarOutput, error) {
	agendamentos, err := uc.repo.ListarPorPrestador(in.UsuarioID)
	if err != nil {
		return nil, err
	}
	return uc.montar(agendamentos, in.Agora)
}

// DoCliente lista os agendamentos feitos pelo cliente.
func (uc *ListarUseCase) DoCliente(in ListarInput) (*ListarOutput, error) {
	agendamentos, err := uc.repo.ListarPorCliente(in.UsuarioID)
	if err != nil {
		return nil, err
	}
	return uc.montar(agendamentos, in.Agora)
}

func (uc *ListarUseCase) montar(agendamentos []*appointment.Appointment, agora time.Time) (*ListarOutput, error) {
	clientes := make(map[string]*client.Client)
	prestadores := make(map[string]*provider.Provider)

	resumos := make([]Resumo, 0, len(agendamentos))
	for _, a := range agendamentos {
		if a.ExpirarSeVencido(agora) {
			if err := uc.repo.Atualizar(a); err != nil {
				return nil, err
			}
		}

		c, err := uc.cliente(a.ClientID, clientes)
		if err != nil {
			return nil, err
		}
		p, err := uc.prestador(a.ProviderID, prestadores)
		if err != nil {
			return nil, err
		}

		resumo := Resumo{
			ID:            a.ID,
			Data:          a.Data,
			InicioMinutos: a.InicioMinutos,
			FimMinutos:    a.FimMinutos,
			Status:        a.Status,
			ExpiraEm:      a.ExpiraEm,
		}
		if c != nil {
			resumo.NomeCliente = c.Nome
			resumo.EmailCliente = c.Email
			resumo.TelefoneCliente = c.Telefone
		}
		if p != nil {
			resumo.NomePrestador = p.Nome
		}
		resumos = append(resumos, resumo)
	}

	return &ListarOutput{Agendamentos: resumos}, nil
}

func (uc *ListarUseCase) cliente(id string, cache map[string]*client.Client) (*client.Client, error) {
	if c, ok := cache[id]; ok {
		return c, nil
	}
	c, err := uc.clientRepo.BuscarPorID(id)
	if err != nil {
		return nil, err
	}
	cache[id] = c
	return c, nil
}

func (uc *ListarUseCase) prestador(id string, cache map[string]*provider.Provider) (*provider.Provider, error) {
	if p, ok := cache[id]; ok {
		return p, nil
	}
	p, err := uc.providerRepo.BuscarPorID(id)
	if err != nil {
		return nil, err
	}
	cache[id] = p
	return p, nil
}
