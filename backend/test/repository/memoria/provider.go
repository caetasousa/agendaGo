package memoria

import (
	"sort"
	"sync"

	"agendago/internal/domain/provider"
	"agendago/internal/pkg/paging"
)

type ProviderMemoria struct {
	mu    sync.RWMutex
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

// AtualizarSenha persiste um novo hash de senha, espelhando o contrato do Postgres.
func (r *ProviderMemoria) AtualizarSenha(id, senhaHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.dados[id]; ok {
		p.SenhaHash = senhaHash
	}
	return nil
}

// Listar devolve uma página de prestadores (inclusive banidos, como a visão de
// moderação) e o total. A ordenação por nome+id espelha o ORDER BY do
// Postgres: sem ela, a ordem viria do mapa — aleatória — e a paginação
// devolveria itens repetidos ou omitidos entre páginas.
func (r *ProviderMemoria) Listar(pag paging.Pagina) ([]*provider.Provider, int, error) {
	todos := r.ordenadosPorNome(func(*provider.Provider) bool { return true })
	return fatiar(todos, pag), len(todos), nil
}

// ListarAtivos devolve uma página de prestadores ativos e o total de ativos —
// a vitrine pública.
func (r *ProviderMemoria) ListarAtivos(pag paging.Pagina) ([]*provider.Provider, int, error) {
	ativos := r.ordenadosPorNome(func(p *provider.Provider) bool { return p.Ativo })
	return fatiar(ativos, pag), len(ativos), nil
}

func (r *ProviderMemoria) ordenadosPorNome(inclui func(*provider.Provider) bool) []*provider.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var selecionados []*provider.Provider
	for _, p := range r.dados {
		if inclui(p) {
			selecionados = append(selecionados, p)
		}
	}
	sort.Slice(selecionados, func(i, j int) bool {
		if selecionados[i].Nome != selecionados[j].Nome {
			return selecionados[i].Nome < selecionados[j].Nome
		}
		return selecionados[i].ID < selecionados[j].ID
	})
	return selecionados
}
