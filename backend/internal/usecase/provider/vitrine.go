package provider

import "agendago/internal/domain/provider"

// repositorioListar busca os prestadores da vitrine.
type repositorioListar interface {
	Listar() ([]*provider.Provider, error)
}

// PrestadorResumo identifica um prestador na vitrine e no link público de
// agendamento. AceitaAgendamentos indica se ele está ofertando horários.
type PrestadorResumo struct {
	ID                        string
	Nome                      string
	DuracaoAtendimentoMinutos int
	AceitaAgendamentos        bool
}

// ListarOutput contém os prestadores da vitrine.
type ListarOutput struct {
	Prestadores []PrestadorResumo
}

// ListarUseCase lista todos os prestadores, para o cliente escolher com quem
// agendar. Quem está com a agenda desativada aparece sem horários.
type ListarUseCase struct {
	repo repositorioListar
}

// NovoListarUseCase cria uma instância de ListarUseCase com o repositório injetado.
func NovoListarUseCase(repo repositorioListar) *ListarUseCase {
	return &ListarUseCase{repo: repo}
}

// Executar devolve os prestadores ativos com os dados mínimos da vitrine.
// Prestadores banidos pelo admin não aparecem na listagem pública.
func (uc *ListarUseCase) Executar() (*ListarOutput, error) {
	todos, err := uc.repo.Listar()
	if err != nil {
		return nil, err
	}

	resumos := make([]PrestadorResumo, 0, len(todos))
	for _, p := range todos {
		if p.Ativo {
			resumos = append(resumos, resumoDe(p))
		}
	}
	return &ListarOutput{Prestadores: resumos}, nil
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
