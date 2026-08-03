package memoria

import (
	"sync"
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/pkg/paging"
)

type AppointmentMemoria struct {
	mu    sync.Mutex
	dados map[string]*appointment.Appointment
}

func NovoAppointmentMemoria() *AppointmentMemoria {
	return &AppointmentMemoria{dados: make(map[string]*appointment.Appointment)}
}

// SalvarSeLivre persiste a solicitação somente se o intervalo não colidir com
// outro agendamento que ocupa horário (anti-overbooking). O mutex faz o papel
// da transação: checagem e escrita são atômicas.
func (r *AppointmentMemoria) SalvarSeLivre(a *appointment.Appointment, agora time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existente := range r.dados {
		if existente.ProviderID != a.ProviderID || !mesmaData(existente.Data, a.Data) {
			continue
		}
		if !existente.Ocupa(agora) {
			continue
		}
		if a.InicioMinutos < existente.FimMinutos && existente.InicioMinutos < a.FimMinutos {
			return appointment.ErrConflitoHorario
		}
	}

	r.dados[a.ID] = a
	return nil
}

// BuscarPorID retorna (nil, nil) quando não há agendamento com o id,
// seguindo o mesmo contrato do repositório Postgres.
func (r *AppointmentMemoria) BuscarPorID(id string) (*appointment.Appointment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a, ok := r.dados[id]; ok {
		return a, nil
	}
	return nil, nil
}

// Atualizar persiste o estado atual do agendamento (status e atualizado_em).
func (r *AppointmentMemoria) Atualizar(a *appointment.Appointment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[a.ID] = a
	return nil
}

// ListarPorPrestador devolve uma página dos agendamentos do prestador, do mais
// recente para o mais antigo, e o total dele.
func (r *AppointmentMemoria) ListarPorPrestador(providerID string, pag paging.Pagina) ([]*appointment.Appointment, int, error) {
	return r.listarPaginado(func(a *appointment.Appointment) bool { return a.ProviderID == providerID }, pag)
}

// ListarPorCliente devolve uma página dos agendamentos do cliente, do mais
// recente para o mais antigo, e o total dele.
func (r *AppointmentMemoria) ListarPorCliente(clientID string, pag paging.Pagina) ([]*appointment.Appointment, int, error) {
	return r.listarPaginado(func(a *appointment.Appointment) bool { return a.ClientID == clientID }, pag)
}

// listarPaginado espelha o repositório Postgres: ordem decrescente por data e
// início (o mais recente primeiro) antes de aplicar a fatia.
func (r *AppointmentMemoria) listarPaginado(filtro func(*appointment.Appointment) bool, pag paging.Pagina) ([]*appointment.Appointment, int, error) {
	todos, err := r.listar(filtro)
	if err != nil {
		return nil, 0, err
	}
	inverter(todos)
	return fatiar(todos, pag), len(todos), nil
}

// inverter reverte a lista já ordenada de forma crescente, produzindo a ordem
// decrescente do ORDER BY ... DESC.
func inverter(as []*appointment.Appointment) {
	for i, j := 0, len(as)-1; i < j; i, j = i+1, j-1 {
		as[i], as[j] = as[j], as[i]
	}
}

// ListarOcupantesPorPeriodo devolve os agendamentos do prestador que ocupam
// horário (SOLICITADO não expirado ou CONFIRMADO) entre as datas, inclusive.
func (r *AppointmentMemoria) ListarOcupantesPorPeriodo(providerID string, de, ate time.Time, agora time.Time) ([]*appointment.Appointment, error) {
	return r.listar(func(a *appointment.Appointment) bool {
		return a.ProviderID == providerID &&
			!a.Data.Before(de) && !a.Data.After(ate) &&
			a.Ocupa(agora)
	})
}

// ListarPorPeriodo devolve todos os agendamentos do prestador entre as datas
// (inclusive), em qualquer status.
func (r *AppointmentMemoria) ListarPorPeriodo(providerID string, de, ate time.Time) ([]*appointment.Appointment, error) {
	return r.listar(func(a *appointment.Appointment) bool {
		return a.ProviderID == providerID &&
			!a.Data.Before(de) && !a.Data.After(ate)
	})
}

// ListarConfirmadosSemLembrete devolve os agendamentos CONFIRMADOs cuja data
// está entre de e ate (inclusive) e cujo lembrete ainda não foi enviado.
func (r *AppointmentMemoria) ListarConfirmadosSemLembrete(de, ate time.Time) ([]*appointment.Appointment, error) {
	return r.listar(func(a *appointment.Appointment) bool {
		return a.Status == appointment.StatusConfirmado &&
			!a.Data.Before(de) && !a.Data.After(ate) &&
			a.LembreteEnviadoEm == nil
	})
}

// MarcarLembreteEnviado marca o lembrete como enviado, mas só se ainda não
// tiver sido, espelhando o claim do repositório Postgres.
func (r *AppointmentMemoria) MarcarLembreteEnviado(id string, quando time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.dados[id]
	if !ok || a.LembreteEnviadoEm != nil {
		return false, nil
	}
	a.LembreteEnviadoEm = &quando
	return true, nil
}

func (r *AppointmentMemoria) listar(filtro func(*appointment.Appointment) bool) ([]*appointment.Appointment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var resultado []*appointment.Appointment
	for _, a := range r.dados {
		if filtro(a) {
			resultado = append(resultado, a)
		}
	}
	ordenarPorDataInicio(resultado)
	return resultado, nil
}

func ordenarPorDataInicio(as []*appointment.Appointment) {
	for i := 1; i < len(as); i++ {
		for j := i; j > 0 && antes(as[j], as[j-1]); j-- {
			as[j], as[j-1] = as[j-1], as[j]
		}
	}
}

func antes(a, b *appointment.Appointment) bool {
	if !mesmaData(a.Data, b.Data) {
		return a.Data.Before(b.Data)
	}
	return a.InicioMinutos < b.InicioMinutos
}

func mesmaData(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}
