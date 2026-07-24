package memoria

import (
	"sync"
	"time"

	"agendago/internal/domain/passwordreset"
)

type PasswordResetMemoria struct {
	mu    sync.Mutex
	dados map[string]*passwordreset.Token
}

func NovoPasswordResetMemoria() *PasswordResetMemoria {
	return &PasswordResetMemoria{dados: make(map[string]*passwordreset.Token)}
}

func (r *PasswordResetMemoria) Salvar(t *passwordreset.Token) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[t.TokenHash] = t
	return nil
}

// Consumir apaga e devolve o token com o hash informado, seguindo o mesmo
// contrato atômico do repositório Postgres. Retorna (nil, nil) quando não
// existe token com o hash.
func (r *PasswordResetMemoria) Consumir(tokenHash string) (*passwordreset.Token, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.dados[tokenHash]
	if !ok {
		return nil, nil
	}
	delete(r.dados, tokenHash)
	return t, nil
}

// RemoverDoUsuario apaga todos os tokens de recuperação pendentes de um usuário.
func (r *PasswordResetMemoria) RemoverDoUsuario(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for hash, t := range r.dados {
		if t.UserID == userID {
			delete(r.dados, hash)
		}
	}
	return nil
}

// RemoverExpirados apaga todos os tokens cuja expira_em já passou.
func (r *PasswordResetMemoria) RemoverExpirados() error {
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
