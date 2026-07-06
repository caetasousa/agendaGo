package repository

import (
	"sync"

	"agendago/internal/domain/client"
)

type ClientMemoria struct {
	mu    sync.RWMutex
	dados map[string]*client.Client
}

func NovoClientMemoria() *ClientMemoria {
	return &ClientMemoria{dados: make(map[string]*client.Client)}
}

func (r *ClientMemoria) Salvar(c *client.Client) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[c.ID] = c
	return nil
}

// BuscarPorEmail retorna (nil, nil) quando não há cliente com o email,
// seguindo o mesmo contrato do repositório Postgres.
func (r *ClientMemoria) BuscarPorEmail(email string) (*client.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.dados {
		if c.Email == email {
			return c, nil
		}
	}
	return nil, nil
}

// BuscarPorID retorna (nil, nil) quando não há cliente com o id, seguindo
// o mesmo contrato do repositório Postgres.
func (r *ClientMemoria) BuscarPorID(id string) (*client.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if c, ok := r.dados[id]; ok {
		return c, nil
	}
	return nil, nil
}
