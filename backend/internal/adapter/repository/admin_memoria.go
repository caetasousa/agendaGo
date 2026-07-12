package repository

import (
	"sync"

	"agendago/internal/domain/admin"
)

type AdminMemoria struct {
	mu    sync.RWMutex
	dados map[string]*admin.Admin
}

func NovoAdminMemoria() *AdminMemoria {
	return &AdminMemoria{dados: make(map[string]*admin.Admin)}
}

// Salvar persiste o admin (upsert por email), seguindo o contrato do Postgres.
func (r *AdminMemoria) Salvar(a *admin.Admin) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, existente := range r.dados {
		if existente.Email == a.Email {
			delete(r.dados, id)
		}
	}
	r.dados[a.ID] = a
	return nil
}

// BuscarPorEmail retorna (nil, nil) quando não há admin com o email.
func (r *AdminMemoria) BuscarPorEmail(email string) (*admin.Admin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, a := range r.dados {
		if a.Email == email {
			return a, nil
		}
	}
	return nil, nil
}

// BuscarPorID retorna (nil, nil) quando não há admin com o id.
func (r *AdminMemoria) BuscarPorID(id string) (*admin.Admin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if a, ok := r.dados[id]; ok {
		return a, nil
	}
	return nil, nil
}
