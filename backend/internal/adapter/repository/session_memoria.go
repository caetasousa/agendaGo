package repository

import (
	"sync"
	"time"

	"agendago/internal/domain/session"
)

type SessionMemoria struct {
	mu    sync.RWMutex
	dados map[string]*session.Session
}

func NovoSessionMemoria() *SessionMemoria {
	return &SessionMemoria{dados: make(map[string]*session.Session)}
}

func (r *SessionMemoria) Salvar(s *session.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[s.TokenHash] = s
	return nil
}

// BuscarPorTokenHash retorna (nil, nil) quando não há sessão com o hash,
// seguindo o mesmo contrato do repositório Postgres.
func (r *SessionMemoria) BuscarPorTokenHash(hash string) (*session.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, ok := r.dados[hash]; ok {
		return s, nil
	}
	return nil, nil
}

// Remover apaga a sessão com o hash informado. Não é erro remover uma sessão inexistente.
func (r *SessionMemoria) Remover(hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.dados, hash)
	return nil
}

// RemoverExpiradas apaga todas as sessões cuja expira_em já passou.
func (r *SessionMemoria) RemoverExpiradas() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	agora := time.Now()
	for hash, s := range r.dados {
		if s.Expirada(agora) {
			delete(r.dados, hash)
		}
	}
	return nil
}
