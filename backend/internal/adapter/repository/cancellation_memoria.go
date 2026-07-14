package repository

import (
	"sync"
	"time"

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

// Remover apaga o token de cancelamento — uso único, chamado depois que o
// cancelamento de fato acontece.
func (r *CancellationMemoria) Remover(hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.dados, hash)
	return nil
}

// RemoverExpirados apaga os tokens de cancelamento cuja expira_em já passou.
func (r *CancellationMemoria) RemoverExpirados() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	agora := time.Now()
	for hash, t := range r.dados {
		if t.Expirado(agora) {
			delete(r.dados, hash)
		}
	}
	return nil
}
