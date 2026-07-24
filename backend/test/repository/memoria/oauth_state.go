package memoria

import (
	"sync"
	"time"

	"agendago/internal/domain/oauthstate"
)

type OAuthStateMemoria struct {
	mu    sync.Mutex
	dados map[string]*oauthstate.State
}

func NovoOAuthStateMemoria() *OAuthStateMemoria {
	return &OAuthStateMemoria{dados: make(map[string]*oauthstate.State)}
}

func (r *OAuthStateMemoria) Salvar(s *oauthstate.State) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[s.StateHash] = s
	return nil
}

// Consumir remove e devolve o state, garantindo uso único, como o repositório Postgres.
func (r *OAuthStateMemoria) Consumir(stateHash string) (*oauthstate.State, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.dados[stateHash]
	if !ok {
		return nil, nil
	}
	delete(r.dados, stateHash)
	return s, nil
}

// RemoverExpirados apaga todos os states cuja expira_em já passou.
func (r *OAuthStateMemoria) RemoverExpirados() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	agora := time.Now()
	for hash, s := range r.dados {
		if s.Expirado(agora) {
			delete(r.dados, hash)
		}
	}
	return nil
}
