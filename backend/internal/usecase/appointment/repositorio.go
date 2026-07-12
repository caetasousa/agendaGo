package appointment

import (
	"errors"
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
)

var (
	// ErrProviderNaoEncontrado é retornado quando o prestador não existe.
	ErrProviderNaoEncontrado = errors.New("prestador não encontrado")
	// ErrClientNaoEncontrado é retornado quando o cliente da sessão não existe mais.
	ErrClientNaoEncontrado = errors.New("cliente não encontrado")
	// ErrClientInativo é retornado quando um cliente banido tenta agendar (por email).
	ErrClientInativo = errors.New("cliente desativado")
	// ErrEmailTemConta é retornado quando o e-mail do convidado pertence a uma
	// conta registrada — sem verificação de posse do e-mail, aceitar criaria
	// agendamentos dentro da conta de um terceiro.
	ErrEmailTemConta = errors.New("este e-mail já tem conta; entre para agendar")
	// ErrAgendamentoNaoEncontrado é retornado quando o agendamento não existe ou não
	// pertence ao usuário — não distingue os casos para não vazar recursos de terceiros.
	ErrAgendamentoNaoEncontrado = errors.New("agendamento não encontrado")
	// ErrHorarioIndisponivel é retornado quando o horário solicitado não é um slot livre.
	ErrHorarioIndisponivel = errors.New("horário indisponível")
	// ErrPeriodoInvalido é retornado quando o período consultado é vazio ou longo demais.
	ErrPeriodoInvalido = errors.New("período inválido")
)

// repositorioAppointment persiste e consulta agendamentos. SalvarSeLivre é a
// barreira anti-overbooking: checagem de conflito e escrita atômicas.
type repositorioAppointment interface {
	SalvarSeLivre(a *appointment.Appointment, agora time.Time) error
	BuscarPorID(id string) (*appointment.Appointment, error)
	Atualizar(a *appointment.Appointment) error
	ListarPorPrestador(providerID string) ([]*appointment.Appointment, error)
	ListarPorCliente(clientID string) ([]*appointment.Appointment, error)
	ListarOcupantesPorPeriodo(providerID string, de, ate time.Time, agora time.Time) ([]*appointment.Appointment, error)
}

// repositorioProvider busca prestadores (duração, buffer e agenda ativa).
type repositorioProvider interface {
	BuscarPorID(id string) (*provider.Provider, error)
}

// repositorioClient busca clientes para validar a sessão e enriquecer
// listagens, e persiste o cliente convidado criado no agendamento sem conta.
type repositorioClient interface {
	BuscarPorID(id string) (*client.Client, error)
	BuscarPorEmail(email string) (*client.Client, error)
	Salvar(c *client.Client) error
}
