package repository

import (
	"sync"
	"time"

	"agendago/internal/domain/availability"
)

type AvailabilityMemoria struct {
	mu       sync.RWMutex
	excecoes map[string]*availability.DateException
}

func NovoAvailabilityMemoria() *AvailabilityMemoria {
	return &AvailabilityMemoria{
		excecoes: make(map[string]*availability.DateException),
	}
}

// BuscarPorData retorna (nil, nil) quando não há definição própria para a
// data, seguindo o mesmo contrato do repositório Postgres.
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

// Listar retorna todas as definições de data do prestador.
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

// SalvarExcecao persiste a definição de uma data, substituindo uma definição
// anterior do mesmo prestador para a mesma data (upsert).
func (r *AvailabilityMemoria) SalvarExcecao(e *availability.DateException) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, existente := range r.excecoes {
		if existente.ProviderID == e.ProviderID && existente.Data.Equal(e.Data) {
			delete(r.excecoes, id)
		}
	}
	r.excecoes[e.ID] = e
	return nil
}

// Remover apaga a definição com o id informado. Não é erro remover uma definição inexistente.
func (r *AvailabilityMemoria) Remover(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.excecoes, id)
	return nil
}
