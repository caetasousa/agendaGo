// Package appointment modela a reserva de um horário entre cliente e
// prestador, com a máquina de estados do ciclo de vida:
//
//	SOLICITADO ──► CONFIRMADO ──► REALIZADO
//	    │              │
//	    │              ├──► NAO_COMPARECEU
//	    │              └──► CANCELADO
//	    ├──► RECUSADO
//	    └──► EXPIRADO
package appointment

import (
	"errors"
	"time"
)

// Status é o estado do agendamento no seu ciclo de vida.
type Status string

const (
	// StatusSolicitado indica que o cliente pediu o horário; ocupa o intervalo até expirar.
	StatusSolicitado Status = "SOLICITADO"
	// StatusConfirmado indica que o prestador aceitou; ocupa o intervalo.
	StatusConfirmado Status = "CONFIRMADO"
	// StatusRealizado indica que o atendimento aconteceu.
	StatusRealizado Status = "REALIZADO"
	// StatusRecusado indica que o prestador negou a solicitação; libera o intervalo.
	StatusRecusado Status = "RECUSADO"
	// StatusExpirado indica que a solicitação venceu o TTL sem confirmação; libera o intervalo.
	StatusExpirado Status = "EXPIRADO"
	// StatusCancelado indica cancelamento por cliente ou prestador; libera o intervalo.
	StatusCancelado Status = "CANCELADO"
	// StatusNaoCompareceu indica que o atendimento confirmado não aconteceu por ausência do cliente.
	StatusNaoCompareceu Status = "NAO_COMPARECEU"
)

var (
	// ErrProviderIDObrigatorio é retornado quando o providerID está vazio.
	ErrProviderIDObrigatorio = errors.New("providerID é obrigatório")
	// ErrClientIDObrigatorio é retornado quando o clientID está vazio.
	ErrClientIDObrigatorio = errors.New("clientID é obrigatório")
	// ErrIntervaloInvalido é retornado quando o intervalo não cabe em um dia ou tem fim antes do início.
	ErrIntervaloInvalido = errors.New("intervalo do agendamento inválido")
	// ErrTransicaoInvalida é retornado quando a mudança de status não existe na máquina de estados.
	ErrTransicaoInvalida = errors.New("transição de status inválida")
	// ErrSolicitacaoExpirada é retornado ao agir sobre uma solicitação que já venceu o TTL.
	ErrSolicitacaoExpirada = errors.New("solicitação expirada")
	// ErrAntecedenciaInsuficiente é retornado ao cancelar dentro da janela mínima antes do início.
	ErrAntecedenciaInsuficiente = errors.New("cancelamento exige antecedência mínima antes do início")
	// ErrAtendimentoNaoIniciado é retornado ao concluir (realizado/não compareceu) antes do horário de início.
	ErrAtendimentoNaoIniciado = errors.New("o horário do atendimento ainda não chegou")
	// ErrConflitoHorario é retornado pela persistência quando o intervalo já
	// está ocupado por outro agendamento (anti-overbooking).
	ErrConflitoHorario = errors.New("o horário já está ocupado")
)

// Appointment representa a reserva de um intervalo de uma data específica.
// Data guarda só o dia; InicioMinutos/FimMinutos são minutos desde a
// meia-noite, no fuso único do sistema.
type Appointment struct {
	ID            string
	ProviderID    string
	ClientID      string
	Data          time.Time
	InicioMinutos int
	FimMinutos    int
	Status        Status
	ExpiraEm      time.Time
	CriadoEm      time.Time
	AtualizadoEm  time.Time
}

// Novo cria uma solicitação de agendamento (SOLICITADO) que já ocupa o
// intervalo, com expiração em agora+ttl. A checagem de conflito com outros
// agendamentos é responsabilidade da persistência, dentro de transação.
func Novo(id, providerID, clientID string, data time.Time, inicioMinutos, fimMinutos int, agora time.Time, ttl time.Duration) (*Appointment, error) {
	if providerID == "" {
		return nil, ErrProviderIDObrigatorio
	}
	if clientID == "" {
		return nil, ErrClientIDObrigatorio
	}
	if inicioMinutos < 0 || fimMinutos > 24*60 || fimMinutos <= inicioMinutos {
		return nil, ErrIntervaloInvalido
	}

	return &Appointment{
		ID:            id,
		ProviderID:    providerID,
		ClientID:      clientID,
		Data:          data,
		InicioMinutos: inicioMinutos,
		FimMinutos:    fimMinutos,
		Status:        StatusSolicitado,
		ExpiraEm:      agora.Add(ttl),
		CriadoEm:      agora,
		AtualizadoEm:  agora,
	}, nil
}

// InicioEm devolve o instante de início do atendimento no fuso informado.
func (a *Appointment) InicioEm(loc *time.Location) time.Time {
	return time.Date(a.Data.Year(), a.Data.Month(), a.Data.Day(), a.InicioMinutos/60, a.InicioMinutos%60, 0, 0, loc)
}

// Ocupa informa se o agendamento segura o intervalo neste instante:
// CONFIRMADO sempre ocupa; SOLICITADO ocupa enquanto não expirar.
func (a *Appointment) Ocupa(agora time.Time) bool {
	switch a.Status {
	case StatusConfirmado:
		return true
	case StatusSolicitado:
		return agora.Before(a.ExpiraEm)
	default:
		return false
	}
}

// ExpirarSeVencido efetiva a expiração lazy: uma solicitação com TTL vencido
// vira EXPIRADO. Devolve true quando houve a transição.
func (a *Appointment) ExpirarSeVencido(agora time.Time) bool {
	if a.Status == StatusSolicitado && !agora.Before(a.ExpiraEm) {
		a.Status = StatusExpirado
		a.AtualizadoEm = agora
		return true
	}
	return false
}

// Confirmar aceita uma solicitação pendente. Retorna ErrSolicitacaoExpirada
// se o TTL venceu e ErrTransicaoInvalida para qualquer outro status.
func (a *Appointment) Confirmar(agora time.Time) error {
	if a.Status != StatusSolicitado {
		return ErrTransicaoInvalida
	}
	if !agora.Before(a.ExpiraEm) {
		return ErrSolicitacaoExpirada
	}
	a.Status = StatusConfirmado
	a.AtualizadoEm = agora
	return nil
}

// Recusar nega uma solicitação pendente, liberando o intervalo.
func (a *Appointment) Recusar(agora time.Time) error {
	if a.Status != StatusSolicitado {
		return ErrTransicaoInvalida
	}
	a.Status = StatusRecusado
	a.AtualizadoEm = agora
	return nil
}

// Cancelar encerra um agendamento CONFIRMADO respeitando a antecedência
// mínima antes do início (no fuso informado). Uma solicitação ainda pendente
// também pode ser cancelada, sem exigência de antecedência — desistir de um
// pedido não confirmado não surpreende ninguém.
func (a *Appointment) Cancelar(agora time.Time, antecedenciaMinima time.Duration, loc *time.Location) error {
	switch a.Status {
	case StatusSolicitado:
		// desistência de pedido pendente: livre
	case StatusConfirmado:
		if a.InicioEm(loc).Sub(agora) < antecedenciaMinima {
			return ErrAntecedenciaInsuficiente
		}
	default:
		return ErrTransicaoInvalida
	}
	a.Status = StatusCancelado
	a.AtualizadoEm = agora
	return nil
}

// MarcarRealizado conclui um agendamento confirmado cujo horário já chegou.
func (a *Appointment) MarcarRealizado(agora time.Time, loc *time.Location) error {
	return a.concluir(StatusRealizado, agora, loc)
}

// MarcarNaoCompareceu registra a ausência do cliente num agendamento
// confirmado cujo horário já chegou.
func (a *Appointment) MarcarNaoCompareceu(agora time.Time, loc *time.Location) error {
	return a.concluir(StatusNaoCompareceu, agora, loc)
}

func (a *Appointment) concluir(destino Status, agora time.Time, loc *time.Location) error {
	if a.Status != StatusConfirmado {
		return ErrTransicaoInvalida
	}
	if agora.Before(a.InicioEm(loc)) {
		return ErrAtendimentoNaoIniciado
	}
	a.Status = destino
	a.AtualizadoEm = agora
	return nil
}
