package repository

import (
	"sync"

	"agendago/internal/domain/cancellation"
)

type CancellationMemoria struct {
	mu    sync.RWMutex
	dados map[string]*cancellation.Token
}

func NovoCancellationMemoria() *CancellationMemoria {
	return &CancellationMemoria{dados: make(map[string]*cancellation.Token)}
}

func (r *CancellationMemoria) Salvar(t *cancellation.Token) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[t.TokenHash] = t
	return nil
}

// BuscarPorTokenHash retorna (nil, nil) quando não há token com o hash,
// seguindo o mesmo contrato do repositório Postgres.
func (r *CancellationMemoria) BuscarPorTokenHash(hash string) (*cancellation.Token, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if t, ok := r.dados[hash]; ok {
		return t, nil
	}
	return nil, nil
}
