package memoria

import (
	"sync"
	"time"

	"agendago/internal/domain/signup"
)

type SignupMemoria struct {
	mu    sync.Mutex
	dados map[string]*signup.Pendente
}

func NovoSignupMemoria() *SignupMemoria {
	return &SignupMemoria{dados: make(map[string]*signup.Pendente)}
}

func (r *SignupMemoria) Salvar(p *signup.Pendente) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[p.TokenHash] = p
	return nil
}

// Consumir apaga e devolve o cadastro pendente com o hash informado, seguindo
// o mesmo contrato atômico do repositório Postgres. (nil, nil) se não existe.
func (r *SignupMemoria) Consumir(tokenHash string) (*signup.Pendente, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.dados[tokenHash]
	if !ok {
		return nil, nil
	}
	delete(r.dados, tokenHash)
	return p, nil
}

// RemoverPorEmail apaga os cadastros pendentes de um email.
func (r *SignupMemoria) RemoverPorEmail(email string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for hash, p := range r.dados {
		if p.Email == email {
			delete(r.dados, hash)
		}
	}
	return nil
}

// RemoverExpirados apaga os cadastros pendentes cuja expira_em já passou.
func (r *SignupMemoria) RemoverExpirados() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	agora := time.Now()
	for hash, p := range r.dados {
		if p.Expirado(agora) {
			delete(r.dados, hash)
		}
	}
	return nil
}
