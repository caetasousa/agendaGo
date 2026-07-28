package memoria

import (
	"sort"
	"sync"
	"time"

	"agendago/internal/domain/convite"
)

type ConviteMemoria struct {
	mu    sync.RWMutex
	dados map[string]*convite.Convite
}

func NovoConviteMemoria() *ConviteMemoria {
	return &ConviteMemoria{dados: make(map[string]*convite.Convite)}
}

// Salvar substitui o convite que existir para o mesmo email e agenda,
// espelhando o ON CONFLICT do repositório Postgres.
func (r *ConviteMemoria) Salvar(c *convite.Convite) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for hash, existente := range r.dados {
		if existente.Email == c.Email && existente.ProviderID == c.ProviderID {
			delete(r.dados, hash)
		}
	}
	r.dados[c.TokenHash] = c
	return nil
}

// BuscarPorTokenHash lê sem consumir, seguindo o contrato do Postgres.
func (r *ConviteMemoria) BuscarPorTokenHash(hash string) (*convite.Convite, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if c, ok := r.dados[hash]; ok {
		return c, nil
	}
	return nil, nil
}

// Consumir apaga ao ler, tornando o token de uso único.
func (r *ConviteMemoria) Consumir(hash string) (*convite.Convite, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.dados[hash]
	if !ok {
		return nil, nil
	}
	delete(r.dados, hash)
	return c, nil
}

// ListarPendentesPorProvider devolve os convites dentro da validade,
// ordenados por criação como o ORDER BY do Postgres.
func (r *ConviteMemoria) ListarPendentesPorProvider(providerID string) ([]*convite.Convite, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agora := time.Now()
	var pendentes []*convite.Convite
	for _, c := range r.dados {
		if c.ProviderID == providerID && !c.Expirado(agora) {
			pendentes = append(pendentes, c)
		}
	}
	sort.Slice(pendentes, func(i, j int) bool {
		if pendentes[i].CriadoEm.Equal(pendentes[j].CriadoEm) {
			return pendentes[i].Email < pendentes[j].Email
		}
		return pendentes[i].CriadoEm.Before(pendentes[j].CriadoEm)
	})
	return pendentes, nil
}

func (r *ConviteMemoria) RemoverPorEmail(providerID, email string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for hash, c := range r.dados {
		if c.ProviderID == providerID && c.Email == email {
			delete(r.dados, hash)
		}
	}
	return nil
}

func (r *ConviteMemoria) RemoverExpirados() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	agora := time.Now()
	for hash, c := range r.dados {
		if c.Expirado(agora) {
			delete(r.dados, hash)
		}
	}
	return nil
}
