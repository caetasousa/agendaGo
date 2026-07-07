package repository

import (
	"sync"

	"agendago/internal/domain/provider"
)

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

// BuscarPorEmail retorna (nil, nil) quando não há prestador com o email,
// seguindo o mesmo contrato do repositório Postgres.
func (r *ProviderMemoria) BuscarPorEmail(email string) (*provider.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.dados {
		if p.Email == email {
			return p, nil
		}
	}
	return nil, nil
}

// BuscarPorID retorna (nil, nil) quando não há prestador com o id, seguindo
// o mesmo contrato do repositório Postgres.
func (r *ProviderMemoria) BuscarPorID(id string) (*provider.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, ok := r.dados[id]; ok {
		return p, nil
	}
	return nil, nil
}

// Atualizar espelha o contrato do Postgres. Como BuscarPorID devolve o
// ponteiro guardado no mapa, o usecase já muta a mesma instância antes de
// chamar Atualizar — reatribuir ao mapa mantém o estado consistente.
func (r *ProviderMemoria) Atualizar(p *provider.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[p.ID] = p
	return nil
}
