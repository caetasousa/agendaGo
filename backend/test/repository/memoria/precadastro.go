package memoria

import (
	"sync"
	"time"

	"agendago/internal/domain/precadastro"
)

type PreCadastroMemoria struct {
	mu    sync.Mutex
	dados map[string]*precadastro.PreCadastro
}

func NovoPreCadastroMemoria() *PreCadastroMemoria {
	return &PreCadastroMemoria{dados: make(map[string]*precadastro.PreCadastro)}
}

func (r *PreCadastroMemoria) Salvar(p *precadastro.PreCadastro) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[p.TokenHash] = p
	return nil
}

// BuscarPorTokenHash retorna (nil, nil) quando não há token com o hash,
// seguindo o mesmo contrato do repositório Postgres. Não apaga o registro —
// a leitura de pré-preenchimento pode acontecer várias vezes antes do submit
// final, que é quem de fato consome o token.
func (r *PreCadastroMemoria) BuscarPorTokenHash(tokenHash string) (*precadastro.PreCadastro, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.dados[tokenHash]
	if !ok {
		return nil, nil
	}
	return p, nil
}

// Consumir retorna (nil, nil) quando não há token com o hash, seguindo o
// mesmo contrato do repositório Postgres. Apaga o registro na leitura — uso
// único.
func (r *PreCadastroMemoria) Consumir(tokenHash string) (*precadastro.PreCadastro, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.dados[tokenHash]
	if !ok {
		return nil, nil
	}
	delete(r.dados, tokenHash)
	return p, nil
}

// RemoverExpirados apaga os tokens de pré-cadastro cuja expira_em já passou.
func (r *PreCadastroMemoria) RemoverExpirados() error {
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
