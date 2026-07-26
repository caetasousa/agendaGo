package provider

import (
	"agendago/internal/domain/provider"
	"agendago/internal/pkg/paging"
)

// repositorioListar busca os prestadores ativos da vitrine — o filtro roda no
// SQL (ver ListarAtivos), não em memória: com LIMIT, filtrar depois de buscar
// devolveria páginas mais curtas que o pedido e esconderia prestadores válidos
// das páginas seguintes.
type repositorioListar interface {
	ListarAtivos(pag paging.Pagina) ([]*provider.Provider, int, error)
}

// PrestadorResumo identifica um prestador na vitrine e no link público de
// agendamento. AceitaAgendamentos indica se ele está ofertando horários.
type PrestadorResumo struct {
	ID                        string
	Nome                      string
	DuracaoAtendimentoMinutos int
	AceitaAgendamentos        bool
}

// ListarOutput contém a página de prestadores da vitrine e o total de ativos.
type ListarOutput struct {
	Prestadores []PrestadorResumo
	Total       int
}

// ListarUseCase lista os prestadores ativos, para o cliente escolher com quem
// agendar. Quem está com a agenda desativada aparece sem horários; quem foi
// banido pelo admin não aparece.
type ListarUseCase struct {
	repo repositorioListar
}

// NovoListarUseCase cria uma instância de ListarUseCase com o repositório injetado.
func NovoListarUseCase(repo repositorioListar) *ListarUseCase {
	return &ListarUseCase{repo: repo}
}

// Executar devolve uma página dos prestadores ativos com os dados mínimos da
// vitrine.
func (uc *ListarUseCase) Executar(pag paging.Pagina) (*ListarOutput, error) {
	ativos, total, err := uc.repo.ListarAtivos(pag)
	if err != nil {
		return nil, err
	}

	resumos := make([]PrestadorResumo, 0, len(ativos))
	for _, p := range ativos {
		resumos = append(resumos, resumoDe(p))
	}
	return &ListarOutput{Prestadores: resumos, Total: total}, nil
}

// BuscarResumoUseCase busca a identificação pública de um prestador — usada
// pela página de agendamento acessada via link direto.
type BuscarResumoUseCase struct {
	repo repositorioPreferencias
}

// NovoBuscarResumoUseCase cria uma instância de BuscarResumoUseCase com o repositório injetado.
func NovoBuscarResumoUseCase(repo repositorioPreferencias) *BuscarResumoUseCase {
	return &BuscarResumoUseCase{repo: repo}
}

// Executar devolve o resumo público do prestador. Retorna
// ErrProviderNaoEncontrado quando o id não existe.
func (uc *BuscarResumoUseCase) Executar(id string) (*PrestadorResumo, error) {
	p, err := uc.repo.BuscarPorID(id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProviderNaoEncontrado
	}
	resumo := resumoDe(p)
	return &resumo, nil
}

func resumoDe(p *provider.Provider) PrestadorResumo {
	return PrestadorResumo{
		ID:                        p.ID,
		Nome:                      p.Nome,
		DuracaoAtendimentoMinutos: p.DuracaoAtendimentoMinutos,
		// Um prestador banido aparece como "não oferta" no link direto, sem
		// vazar o motivo — a página pública simplesmente não mostra horários.
		AceitaAgendamentos: p.Ativo && p.AceitaAgendamentos,
	}
}
