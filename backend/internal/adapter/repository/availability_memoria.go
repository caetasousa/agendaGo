package repository

import (
	"sync"
	"time"

	"agendago/internal/domain/availability"
)

type AvailabilityMemoria struct {
	mu        sync.RWMutex
	schedules map[string]*availability.WeeklySchedule
	excecoes  map[string]*availability.DateException
}

func NovoAvailabilityMemoria() *AvailabilityMemoria {
	return &AvailabilityMemoria{
		schedules: make(map[string]*availability.WeeklySchedule),
		excecoes:  make(map[string]*availability.DateException),
	}
}

// Buscar retorna (nil, nil) quando o prestador nunca configurou a grade
// semanal, seguindo o mesmo contrato do repositório Postgres.
func (r *AvailabilityMemoria) Buscar(providerID string) (*availability.WeeklySchedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, ok := r.schedules[providerID]; ok {
		return s, nil
	}
	return nil, nil
}

// Salvar substitui a grade semanal do prestador por completo.
func (r *AvailabilityMemoria) Salvar(s *availability.WeeklySchedule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.schedules[s.ProviderID] = s
	return nil
}

// BuscarPorData retorna (nil, nil) quando não há exceção para a data,
// seguindo o mesmo contrato do repositório Postgres.
func (r *AvailabilityMemoria) BuscarPorData(providerID string, data time.Time) (*availability.DateException, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, e := range r.excecoes {
		if e.ProviderID == providerID && e.Data.Equal(data) {
			return e, nil
		}
	}
	return nil, nil
}

// BuscarPorID retorna (nil, nil) quando não há exceção com o id.
func (r *AvailabilityMemoria) BuscarPorID(id string) (*availability.DateException, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if e, ok := r.excecoes[id]; ok {
		return e, nil
	}
	return nil, nil
}

// Listar retorna todas as exceções de data do prestador.
func (r *AvailabilityMemoria) Listar(providerID string) ([]*availability.DateException, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var excecoes []*availability.DateException
	for _, e := range r.excecoes {
		if e.ProviderID == providerID {
			excecoes = append(excecoes, e)
		}
	}
	return excecoes, nil
}

// SalvarExcecao persiste uma nova exceção de data.
func (r *AvailabilityMemoria) SalvarExcecao(e *availability.DateException) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.excecoes[e.ID] = e
	return nil
}

// Remover apaga a exceção com o id informado. Não é erro remover uma exceção inexistente.
func (r *AvailabilityMemoria) Remover(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.excecoes, id)
	return nil
}
