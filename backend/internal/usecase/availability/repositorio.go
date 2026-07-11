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
	// ErrDiaNaoDefinido é retornado ao remover a definição de uma data que não tem definição própria.
	ErrDiaNaoDefinido = errors.New("não há definição própria para esta data")
	// ErrPeriodoInvalido é retornado quando o período consultado é vazio ou longo demais.
	ErrPeriodoInvalido = errors.New("período inválido")
)

// repositorioDateException busca, persiste (upsert por data), lista e remove
// as definições próprias de data do prestador.
type repositorioDateException interface {
	BuscarPorData(providerID string, data time.Time) (*availability.DateException, error)
	Listar(providerID string) ([]*availability.DateException, error)
	SalvarExcecao(e *availability.DateException) error
	Remover(id string) error
}

// repositorioProvider busca o prestador para checar AceitaAgendamentos.
type repositorioProvider interface {
	BuscarPorID(id string) (*provider.Provider, error)
}
