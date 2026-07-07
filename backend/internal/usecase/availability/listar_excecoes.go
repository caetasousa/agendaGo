package availability

import (
	"time"

	"agendago/internal/domain/availability"
)

// ListarExcecoesInput identifica o prestador cujas exceções serão listadas.
type ListarExcecoesInput struct {
	ProviderID string
}

// ExcecaoResumo contém os dados de uma exceção de data para exibição.
type ExcecaoResumo struct {
	ID     string
	Data   time.Time
	Tipo   availability.TipoExcecao
	Blocos []availability.TimeBlock
}

// ListarExcecoesOutput contém as exceções de data do prestador.
type ListarExcecoesOutput struct {
	Excecoes []ExcecaoResumo
}

// ListarExcecoesUseCase lista todas as exceções de data cadastradas pelo prestador.
type ListarExcecoesUseCase struct {
	repo repositorioDateException
}

// NovoListarExcecoesUseCase cria uma instância de ListarExcecoesUseCase com o repositório injetado.
func NovoListarExcecoesUseCase(repo repositorioDateException) *ListarExcecoesUseCase {
	return &ListarExcecoesUseCase{repo: repo}
}

// Executar lista as exceções do prestador.
func (uc *ListarExcecoesUseCase) Executar(in ListarExcecoesInput) (*ListarExcecoesOutput, error) {
	excecoes, err := uc.repo.Listar(in.ProviderID)
	if err != nil {
		return nil, err
	}

	resumos := make([]ExcecaoResumo, 0, len(excecoes))
	for _, e := range excecoes {
		resumos = append(resumos, ExcecaoResumo{ID: e.ID, Data: e.Data, Tipo: e.Tipo, Blocos: e.Blocos})
	}

	return &ListarExcecoesOutput{Excecoes: resumos}, nil
}
