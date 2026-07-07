package availability

import (
	"time"

	"agendago/internal/domain/availability"

	"github.com/google/uuid"
)

// CriarExcecaoInput contém os dados para criar uma exceção de data. ProviderID
// vem da identidade da sessão autenticada, nunca do corpo da requisição.
type CriarExcecaoInput struct {
	ProviderID string
	Data       time.Time
	Tipo       availability.TipoExcecao
	Blocos     []BlocoInput
}

// CriarExcecaoOutput contém os dados da exceção criada.
type CriarExcecaoOutput struct {
	ID     string
	Data   time.Time
	Tipo   availability.TipoExcecao
	Blocos []availability.TimeBlock
}

// CriarExcecaoUseCase cria uma exceção de data (bloqueio ou extra) para o prestador.
type CriarExcecaoUseCase struct {
	repo repositorioDateException
}

// NovoCriarExcecaoUseCase cria uma instância de CriarExcecaoUseCase com o repositório injetado.
func NovoCriarExcecaoUseCase(repo repositorioDateException) *CriarExcecaoUseCase {
	return &CriarExcecaoUseCase{repo: repo}
}

// Executar valida que não existe outra exceção para a mesma data, monta os
// blocos (se for do tipo extra) e persiste. Retorna ErrExcecaoJaExiste se já
// houver uma exceção na data.
func (uc *CriarExcecaoUseCase) Executar(in CriarExcecaoInput) (*CriarExcecaoOutput, error) {
	existente, err := uc.repo.BuscarPorData(in.ProviderID, in.Data)
	if err != nil {
		return nil, err
	}
	if existente != nil {
		return nil, ErrExcecaoJaExiste
	}

	var blocos []availability.TimeBlock
	for _, b := range in.Blocos {
		bloco, err := availability.NovoTimeBlock(b.InicioMinutos, b.FimMinutos)
		if err != nil {
			return nil, err
		}
		blocos = append(blocos, bloco)
	}

	excecao, err := availability.NovaDateException(uuid.NewString(), in.ProviderID, in.Data, in.Tipo, blocos)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.SalvarExcecao(excecao); err != nil {
		return nil, err
	}

	return &CriarExcecaoOutput{
		ID:     excecao.ID,
		Data:   excecao.Data,
		Tipo:   excecao.Tipo,
		Blocos: excecao.Blocos,
	}, nil
}
