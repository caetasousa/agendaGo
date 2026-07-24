package memoria

import (
	"sync"

	"agendago/internal/domain/socialidentity"
)

type SocialIdentityMemoria struct {
	mu    sync.RWMutex
	dados map[string]*socialidentity.Identidade // chave: provedor+"|"+sub
}

func NovoSocialIdentityMemoria() *SocialIdentityMemoria {
	return &SocialIdentityMemoria{dados: make(map[string]*socialidentity.Identidade)}
}

func (r *SocialIdentityMemoria) Salvar(i *socialidentity.Identidade) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[chaveIdentidade(i.Provedor, i.Sub)] = i
	return nil
}

// BuscarPorProvedorSub retorna (nil, nil) quando não há identidade com o
// (provedor, sub), seguindo o mesmo contrato do repositório Postgres.
func (r *SocialIdentityMemoria) BuscarPorProvedorSub(provedor socialidentity.Provedor, sub string) (*socialidentity.Identidade, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if i, ok := r.dados[chaveIdentidade(provedor, sub)]; ok {
		return i, nil
	}
	return nil, nil
}

// RemoverDoUsuario apaga todas as identidades sociais de um usuário.
func (r *SocialIdentityMemoria) RemoverDoUsuario(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for chave, i := range r.dados {
		if i.UserID == userID {
			delete(r.dados, chave)
		}
	}
	return nil
}

func chaveIdentidade(provedor socialidentity.Provedor, sub string) string {
	return string(provedor) + "|" + sub
}
