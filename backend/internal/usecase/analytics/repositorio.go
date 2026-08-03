package analytics

import (
	"errors"
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/domain/ocupacao"
	ucavailability "agendago/internal/usecase/availability"
)

// ErrPeriodoInvalido é retornado quando o período pedido está invertido ou é
// maior que maxDiasPeriodo.
var ErrPeriodoInvalido = errors.New("período inválido")

// resolvedorAgenda resolve o expediente de cada dia do período — implementado
// por availability.ConsultarAgendaUseCase. A regra de qual expediente vale em
// qual data (definição própria, padrão, dono banido) mora lá e não é
// reimplementada aqui.
type resolvedorAgenda interface {
	Executar(in ucavailability.ConsultarAgendaInput) (*ucavailability.ConsultarAgendaOutput, error)
}

// repositorioAppointment lê os agendamentos do período em qualquer status —
// o funil precisa dos desfechos, não só de quem ocupa horário.
type repositorioAppointment interface {
	ListarPorPeriodo(providerID string, de, ate time.Time) ([]*appointment.Appointment, error)
}

// repositorioOcupacao lê os compromissos pessoais do prestador no período.
type repositorioOcupacao interface {
	ListarPorPeriodo(providerID string, de, ate time.Time) ([]*ocupacao.Ocupacao, error)
}
