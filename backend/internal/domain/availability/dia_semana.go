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

// DiaSemanaDe converte um time.Time (já no fuso correto) para DiaSemana.
func DiaSemanaDe(data time.Time) DiaSemana {
	return DiaSemana(data.Weekday())
}
