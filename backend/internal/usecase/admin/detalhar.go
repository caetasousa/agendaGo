package admin

import (
	"time"

	ucappointment "agendago/internal/usecase/appointment"
)

// listadorAgendamentos devolve os agendamentos de um usuário como resumos —
// implementado por appointment.ListarUseCase, reaproveitando a expiração lazy
// e os nomes/contato já resolvidos.
type listadorAgendamentos interface {
	DoPrestador(in ucappointment.ListarInput) (*ucappointment.ListarOutput, error)
	DoCliente(in ucappointment.ListarInput) (*ucappointment.ListarOutput, error)
}

// AgendamentoResumo é um agendamento na visão de detalhe do admin.
type AgendamentoResumo struct {
	ID              string
	Data            time.Time
	InicioMinutos   int
	FimMinutos      int
	Status          string
	NomeCliente     string
	EmailCliente    string
	TelefoneCliente string
	NomePrestador   string
}

// DetalhePrestador reúne os dados do prestador e os agendamentos que ele recebeu.
type DetalhePrestador struct {
	ID                 string
	Nome               string
	Email              string
	Ativo              bool
	AceitaAgendamentos bool
	DescansoMinutos    int
	DuracaoMinutos     int
	Agendamentos       []AgendamentoResumo
}

// DetalheCliente reúne os dados do cliente e os agendamentos que ele fez.
type DetalheCliente struct {
	ID           string
	Nome         string
	Email        string
	Telefone     string
	Ativo        bool
	TemConta     bool
	Agendamentos []AgendamentoResumo
}

// DetalharUseCase monta o detalhe em leitura de um prestador ou cliente para o
// admin: dados cadastrais + os agendamentos daquele usuário.
type DetalharUseCase struct {
	providers    repositorioProvider
	clients      repositorioClient
	agendamentos listadorAgendamentos
}

// NovoDetalharUseCase cria uma instância de DetalharUseCase com as dependências injetadas.
func NovoDetalharUseCase(providers repositorioProvider, clients repositorioClient, agendamentos listadorAgendamentos) *DetalharUseCase {
	return &DetalharUseCase{providers: providers, clients: clients, agendamentos: agendamentos}
}

// Prestador devolve o detalhe do prestador. Retorna ErrProviderNaoEncontrado
// se o id não existe.
func (uc *DetalharUseCase) Prestador(id string, agora time.Time) (*DetalhePrestador, error) {
	p, err := uc.providers.BuscarPorID(id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProviderNaoEncontrado
	}

	out, err := uc.agendamentos.DoPrestador(ucappointment.ListarInput{UsuarioID: id, Agora: agora})
	if err != nil {
		return nil, err
	}

	return &DetalhePrestador{
		ID:                 p.ID,
		Nome:               p.Nome,
		Email:              p.Email,
		Ativo:              p.Ativo,
		AceitaAgendamentos: p.AceitaAgendamentos,
		DescansoMinutos:    p.DescansoMinutos,
		DuracaoMinutos:     p.DuracaoAtendimentoMinutos,
		Agendamentos:       paraResumos(out),
	}, nil
}

// Cliente devolve o detalhe do cliente. Retorna ErrClientNaoEncontrado se o id
// não existe.
func (uc *DetalharUseCase) Cliente(id string, agora time.Time) (*DetalheCliente, error) {
	c, err := uc.clients.BuscarPorID(id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrClientNaoEncontrado
	}

	out, err := uc.agendamentos.DoCliente(ucappointment.ListarInput{UsuarioID: id, Agora: agora})
	if err != nil {
		return nil, err
	}

	return &DetalheCliente{
		ID:           c.ID,
		Nome:         c.Nome,
		Email:        c.Email,
		Telefone:     c.Telefone,
		Ativo:        c.Ativo,
		TemConta:     c.TemConta(),
		Agendamentos: paraResumos(out),
	}, nil
}

func paraResumos(out *ucappointment.ListarOutput) []AgendamentoResumo {
	resumos := make([]AgendamentoResumo, 0, len(out.Agendamentos))
	for _, a := range out.Agendamentos {
		resumos = append(resumos, AgendamentoResumo{
			ID:              a.ID,
			Data:            a.Data,
			InicioMinutos:   a.InicioMinutos,
			FimMinutos:      a.FimMinutos,
			Status:          string(a.Status),
			NomeCliente:     a.NomeCliente,
			EmailCliente:    a.EmailCliente,
			TelefoneCliente: a.TelefoneCliente,
			NomePrestador:   a.NomePrestador,
		})
	}
	return resumos
}
