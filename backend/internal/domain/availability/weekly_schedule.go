package availability

import (
	"errors"
	"time"
)

// DiaSemana replica os valores de time.Weekday (0=domingo..6=sábado) para não
// acoplar o domínio ao pacote time além do necessário.
type DiaSemana int

const (
	Domingo DiaSemana = 0
	Segunda DiaSemana = 1
	Terca   DiaSemana = 2
	Quarta  DiaSemana = 3
	Quinta  DiaSemana = 4
	Sexta   DiaSemana = 5
	Sabado  DiaSemana = 6
)

// ErrProviderIDObrigatorio é retornado quando o providerID está vazio.
var ErrProviderIDObrigatorio = errors.New("providerID é obrigatório")

// WeeklySchedule é a grade semanal recorrente de um prestador: um conjunto de
// TimeBlock por dia da semana. Dias sem blocos configurados significam "não
// trabalha" naquele dia.
type WeeklySchedule struct {
	ProviderID string
	Dias       map[DiaSemana][]TimeBlock
}

// NovaWeeklySchedule cria a grade semanal a partir dos blocos por dia,
// normalizando (ordenando e mesclando adjacentes) cada dia independentemente.
// Retorna erro do primeiro dia com blocos sobrepostos ou inválidos.
func NovaWeeklySchedule(providerID string, blocosPorDia map[DiaSemana][]TimeBlock) (*WeeklySchedule, error) {
	if providerID == "" {
		return nil, ErrProviderIDObrigatorio
	}

	dias := make(map[DiaSemana][]TimeBlock, len(blocosPorDia))
	for dia, blocos := range blocosPorDia {
		normalizados, err := normalizarBlocos(blocos)
		if err != nil {
			return nil, err
		}
		if len(normalizados) > 0 {
			dias[dia] = normalizados
		}
	}

	return &WeeklySchedule{ProviderID: providerID, Dias: dias}, nil
}

// BlocosDoDia devolve os blocos configurados para o dia da semana informado
// (vazio se o prestador não trabalha nesse dia).
func (w *WeeklySchedule) BlocosDoDia(dia DiaSemana) []TimeBlock {
	return w.Dias[dia]
}

// DiaSemanaDe converte um time.Time (já no fuso correto) para DiaSemana.
func DiaSemanaDe(data time.Time) DiaSemana {
	return DiaSemana(data.Weekday())
}
