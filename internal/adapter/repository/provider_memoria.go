package repository

import (
	"errors"
	"sync"

	"agendago/internal/domain/provider"
)

var ErrProviderNaoEncontrado = errors.New("prestador não encontrado")

type ProviderMemoria struct {
	mu   sync.RWMutex
	dados map[string]*provider.Provider
}

func NovoProviderMemoria() *ProviderMemoria {
	return &ProviderMemoria{dados: make(map[string]*provider.Provider)}
}

func (r *ProviderMemoria) Salvar(p *provider.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[p.ID] = p
	return nil
}

func (r *ProviderMemoria) BuscarPorEmail(email string) (*provider.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.dados {
		if p.Email == email {
			return p, nil
		}
	}
	return nil, ErrProviderNaoEncontrado
}

func (r *ProviderMemoria) BuscarPorID(id string) (*provider.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.dados[id]
	if !ok {
		return nil, ErrProviderNaoEncontrado
	}
	return p, nil
}
