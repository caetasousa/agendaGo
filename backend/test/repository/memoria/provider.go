package memoria

import (
	"sort"
	"strconv"
	"sync"

	"agendago/internal/domain/provider"
	"agendago/internal/pkg/paging"
)

type ProviderMemoria struct {
	mu    sync.RWMutex
	dados map[string]*provider.Provider
	// membros responde quem é o dono e se ele está ativo — no Postgres isso é
	// um join. Pode ser nil nos testes que não exercitam a vitrine.
	membros *MembroMemoria
}

func NovoProviderMemoria() *ProviderMemoria {
	return &ProviderMemoria{dados: make(map[string]*provider.Provider)}
}

// NovoProviderMemoriaCom liga o fake de agendas ao de vínculos, necessário
// para ListarAtivos filtrar como o join do Postgres filtra.
func NovoProviderMemoriaCom(membros *MembroMemoria) *ProviderMemoria {
	return &ProviderMemoria{dados: make(map[string]*provider.Provider), membros: membros}
}

// Salvar persiste o prestador, desempatando o slug como o Postgres faz: se
// "joao-silva" já existe em outro prestador, vira "joao-silva-2". Sem isto um
// teste em memória passaria onde o banco recusaria por violação de unicidade.
func (r *ProviderMemoria) Salvar(p *provider.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if p.Slug != "" {
		base := p.Slug
		for tentativa := 2; r.slugTomado(p.Slug, p.ID); tentativa++ {
			p.Slug = base + "-" + strconv.Itoa(tentativa)
		}
	}
	r.dados[p.ID] = p
	return nil
}

// slugTomado informa se OUTRO prestador já usa o slug.
func (r *ProviderMemoria) slugTomado(slug, excetoID string) bool {
	for _, existente := range r.dados {
		if existente.ID != excetoID && existente.Slug == slug {
			return true
		}
	}
	return false
}

// BuscarPorSlug retorna (nil, nil) quando não há prestador com o slug,
// seguindo o mesmo contrato do repositório Postgres.
func (r *ProviderMemoria) BuscarPorSlug(slug string) (*provider.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.dados {
		if p.Slug == slug {
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

// Listar devolve uma página de prestadores (inclusive banidos, como a visão de
// moderação) e o total. A ordenação por nome+id espelha o ORDER BY do
// Postgres: sem ela, a ordem viria do mapa — aleatória — e a paginação
// devolveria itens repetidos ou omitidos entre páginas.
func (r *ProviderMemoria) Listar(pag paging.Pagina) ([]*provider.Provider, int, error) {
	todos := r.ordenadosPorNome(func(*provider.Provider) bool { return true })
	return fatiar(todos, pag), len(todos), nil
}

// ListarAtivos devolve uma página das agendas cujo dono está ativo e o total
// delas — a vitrine pública. O banimento mora em usuarios, então o fake
// precisa do de vínculos para responder como o join do Postgres responderia;
// sem ele, nenhuma agenda é considerada ativa.
func (r *ProviderMemoria) ListarAtivos(pag paging.Pagina) ([]*provider.Provider, int, error) {
	ativos := r.ordenadosPorNome(func(p *provider.Provider) bool {
		if r.membros == nil {
			return false
		}
		ativo, err := r.membros.DonoAtivo(p.ID)
		return err == nil && ativo
	})
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
