package availability

import (
	"errors"
	"time"

	"agendago/internal/domain/availability"
	"agendago/internal/domain/provider"
)

var (
	// ErrProviderNaoEncontrado é retornado quando o prestador da sessão não existe mais.
	ErrProviderNaoEncontrado = errors.New("prestador não encontrado")
	// ErrExcecaoNaoEncontrada é retornado quando a exceção não existe ou não pertence ao prestador.
	ErrExcecaoNaoEncontrada = errors.New("exceção não encontrada")
	// ErrExcecaoJaExiste é retornado quando já existe uma exceção para a data informada.
	ErrExcecaoJaExiste = errors.New("já existe uma exceção para esta data")
)

// repositorioWeeklySchedule busca e persiste a grade semanal completa do prestador.
type repositorioWeeklySchedule interface {
	Buscar(providerID string) (*availability.WeeklySchedule, error)
	Salvar(s *availability.WeeklySchedule) error
}

// repositorioDateException busca, cria, lista e remove exceções de data.
type repositorioDateException interface {
	BuscarPorData(providerID string, data time.Time) (*availability.DateException, error)
	BuscarPorID(id string) (*availability.DateException, error)
	Listar(providerID string) ([]*availability.DateException, error)
	SalvarExcecao(e *availability.DateException) error
	Remover(id string) error
}

// repositorioProvider busca o prestador para checar AceitaAgendamentos.
type repositorioProvider interface {
	BuscarPorID(id string) (*provider.Provider, error)
}
