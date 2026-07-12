package repository

import (
	"sync"
	"time"

	"agendago/internal/domain/appointment"
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

// ListarPorPrestador devolve os agendamentos do prestador ordenados por data e início.
func (r *AppointmentMemoria) ListarPorPrestador(providerID string) ([]*appointment.Appointment, error) {
	return r.listar(func(a *appointment.Appointment) bool { return a.ProviderID == providerID })
}

// ListarPorCliente devolve os agendamentos do cliente ordenados por data e início.
func (r *AppointmentMemoria) ListarPorCliente(clientID string) ([]*appointment.Appointment, error) {
	return r.listar(func(a *appointment.Appointment) bool { return a.ClientID == clientID })
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
