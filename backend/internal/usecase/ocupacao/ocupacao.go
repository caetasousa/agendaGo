// Package ocupacao contém os usecases de compromisso pessoal do prestador:
// criar, listar e remover intervalos que deixam de ser ofertados.
package ocupacao

import (
	"errors"
	"time"

	"agendago/internal/domain/appointment"
	domocupacao "agendago/internal/domain/ocupacao"
)

var (
	// ErrOcupacaoNaoEncontrada é retornado quando o compromisso não existe, e
	// também quando existe mas pertence a outra agenda: quem não é dono não
	// precisa saber que o id existe.
	ErrOcupacaoNaoEncontrada = errors.New("compromisso não encontrado")
	// ErrPeriodoInvalido é retornado para período invertido ou longo demais.
	ErrPeriodoInvalido = errors.New("período inválido")
)

// maxDiasPeriodo limita a consulta, como nas demais rotas de agenda.
const maxDiasPeriodo = 92

// repositorioOcupacao persiste e consulta compromissos.
type repositorioOcupacao interface {
	Salvar(o *domocupacao.Ocupacao) error
	ListarPorPeriodo(providerID string, de, ate time.Time) ([]*domocupacao.Ocupacao, error)
	BuscarPorID(id string) (*domocupacao.Ocupacao, error)
	Remover(id string) error
}

// repositorioAppointment lista os agendamentos que ocupam horário — é o que
// impede criar compromisso por cima de cliente já marcado.
type repositorioAppointment interface {
	ListarOcupantesPorPeriodo(providerID string, de, ate time.Time, agora time.Time) ([]*appointment.Appointment, error)
}
