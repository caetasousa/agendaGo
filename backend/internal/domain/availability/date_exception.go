package availability

import (
	"errors"
	"time"
)

// TipoExcecao identifica se uma exceção de data deixa o dia inteiro
// indisponível ou substitui o expediente padrão por horários personalizados.
type TipoExcecao string

const (
	// TipoBloqueio marca a data como indisponível o dia inteiro.
	TipoBloqueio TipoExcecao = "bloqueio"
	// TipoExtra substitui o expediente padrão da data por horários personalizados.
	TipoExtra TipoExcecao = "extra"
)

var (
	// ErrTipoInvalido é retornado quando TipoExcecao não é "bloqueio" nem "extra".
	ErrTipoInvalido = errors.New("tipo de exceção inválido")
	// ErrBloqueioComBlocos é retornado quando uma exceção de bloqueio recebe blocos de horário.
	ErrBloqueioComBlocos = errors.New("exceção de bloqueio não pode ter blocos de horário")
	// ErrExtraSemBlocos é retornado quando uma exceção extra não recebe nenhum bloco de horário.
	ErrExtraSemBlocos = errors.New("exceção extra exige ao menos um bloco de horário")
)

// DateException representa a definição própria de uma data, sobrepondo o
// expediente padrão do prestador. BLOQUEIO nunca carrega Blocos (dia inteiro
// indisponível); EXTRA sempre carrega ao menos um bloco, validado com as
// regras usuais (sem overlap, granularidade, merge de adjacentes).
type DateException struct {
	ID         string
	ProviderID string
	Data       time.Time
	Tipo       TipoExcecao
	Blocos     []TimeBlock
	CriadoEm   time.Time
}

// NovaDateException cria uma exceção de data. Para TipoBloqueio, blocos deve
// vir vazio (ErrBloqueioComBlocos caso contrário). Para TipoExtra, blocos deve
// ter ao menos um item (ErrExtraSemBlocos caso contrário) e é normalizado
// (ordenado e mesclado, com a mesma regra de overlap da grade semanal).
func NovaDateException(id, providerID string, data time.Time, tipo TipoExcecao, blocos []TimeBlock) (*DateException, error) {
	if providerID == "" {
		return nil, ErrProviderIDObrigatorio
	}
	if tipo != TipoBloqueio && tipo != TipoExtra {
		return nil, ErrTipoInvalido
	}

	var blocosFinais []TimeBlock
	switch tipo {
	case TipoBloqueio:
		if len(blocos) > 0 {
			return nil, ErrBloqueioComBlocos
		}
	case TipoExtra:
		if len(blocos) == 0 {
			return nil, ErrExtraSemBlocos
		}
		normalizados, err := normalizarBlocos(blocos)
		if err != nil {
			return nil, err
		}
		blocosFinais = normalizados
	}

	return &DateException{
		ID:         id,
		ProviderID: providerID,
		Data:       data,
		Tipo:       tipo,
		Blocos:     blocosFinais,
		CriadoEm:   time.Now(),
	}, nil
}
