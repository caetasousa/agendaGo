package availability

import (
	"errors"
	"sort"
)

// GranularidadeMinutos é o múltiplo mínimo exigido para início e fim de um bloco.
const GranularidadeMinutos = 15

// MinutosPorDia é o limite superior exclusivo de um horário dentro de um dia
// (24h * 60min) — usado para proibir blocos que cruzem a meia-noite.
const MinutosPorDia = 24 * 60

var (
	// ErrFimAntesDoInicio é retornado quando o fim do bloco não é estritamente posterior ao início.
	ErrFimAntesDoInicio = errors.New("fim deve ser posterior ao início")
	// ErrForaDoDia é retornado quando início ou fim ficam fora do intervalo [0, 1440] do dia.
	ErrForaDoDia = errors.New("bloco não pode cruzar a meia-noite")
	// ErrGranularidadeInvalida é retornado quando início ou fim não são múltiplos de 15 minutos.
	ErrGranularidadeInvalida = errors.New("horário deve ser múltiplo de 15 minutos")
	// ErrBlocosSobrepostos é retornado quando dois blocos do mesmo conjunto se sobrepõem de fato.
	ErrBlocosSobrepostos = errors.New("blocos não podem se sobrepor")
)

// TimeBlock representa um intervalo de tempo dentro de um único dia, em
// minutos desde a meia-noite (0 = 00:00, 1440 = 24:00). InicioMinutos e
// FimMinutos nunca cruzam a fronteira do dia — um expediente noturno deve
// ser partido em dois TimeBlock, um por dia.
type TimeBlock struct {
	InicioMinutos int
	FimMinutos    int
}

// NovoTimeBlock valida e cria um bloco de horário. Retorna erro se fim não
// for posterior a início, se início/fim caírem fora de [0, 1440], ou se
// início/fim não forem múltiplos de GranularidadeMinutos.
func NovoTimeBlock(inicioMinutos, fimMinutos int) (TimeBlock, error) {
	if inicioMinutos < 0 || inicioMinutos >= MinutosPorDia || fimMinutos < 0 || fimMinutos > MinutosPorDia {
		return TimeBlock{}, ErrForaDoDia
	}
	if fimMinutos <= inicioMinutos {
		return TimeBlock{}, ErrFimAntesDoInicio
	}
	if inicioMinutos%GranularidadeMinutos != 0 || fimMinutos%GranularidadeMinutos != 0 {
		return TimeBlock{}, ErrGranularidadeInvalida
	}
	return TimeBlock{InicioMinutos: inicioMinutos, FimMinutos: fimMinutos}, nil
}

// NormalizarBlocos ordena os blocos por início e mescla somente os que são
// exatamente adjacentes (fim de um igual ao início do próximo). Overlap
// franco — blocos que se cruzam sem serem meramente adjacentes — é
// considerado erro do usuário e retorna ErrBlocosSobrepostos. Reaproveitada
// por qualquer conjunto de blocos de um único dia (grade de exceção, ou o
// expediente padrão configurável do prestador).
func NormalizarBlocos(blocos []TimeBlock) ([]TimeBlock, error) {
	return normalizarBlocos(blocos)
}

func normalizarBlocos(blocos []TimeBlock) ([]TimeBlock, error) {
	if len(blocos) == 0 {
		return nil, nil
	}

	ordenados := append([]TimeBlock(nil), blocos...)
	sort.Slice(ordenados, func(i, j int) bool {
		return ordenados[i].InicioMinutos < ordenados[j].InicioMinutos
	})

	mesclados := []TimeBlock{ordenados[0]}
	for _, atual := range ordenados[1:] {
		ultimo := &mesclados[len(mesclados)-1]
		switch {
		case atual.InicioMinutos == ultimo.FimMinutos:
			ultimo.FimMinutos = atual.FimMinutos
		case atual.InicioMinutos < ultimo.FimMinutos:
			return nil, ErrBlocosSobrepostos
		default:
			mesclados = append(mesclados, atual)
		}
	}
	return mesclados, nil
}
