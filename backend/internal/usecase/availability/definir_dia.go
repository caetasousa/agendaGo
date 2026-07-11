package availability

import (
	"time"

	"agendago/internal/domain/availability"

	"github.com/google/uuid"
)

// BlocoInput representa um bloco de horário em minutos, ainda não validado pelo domínio.
type BlocoInput struct {
	InicioMinutos int
	FimMinutos    int
}

// DefinirDiaInput contém a definição própria de uma data: bloqueio (dia
// indisponível) ou extra (horários personalizados). ProviderID vem da
// identidade da sessão autenticada, nunca do corpo da requisição.
type DefinirDiaInput struct {
	ProviderID string
	Data       time.Time
	Tipo       availability.TipoExcecao
	Blocos     []BlocoInput
}

// DefinirDiaOutput contém a definição persistida da data.
type DefinirDiaOutput struct {
	Data   time.Time
	Tipo   availability.TipoExcecao
	Blocos []availability.TimeBlock
}

// DefinirDiaUseCase cria ou substitui a definição própria de uma data do
// prestador (upsert por data).
type DefinirDiaUseCase struct {
	repo repositorioDateException
}

// NovoDefinirDiaUseCase cria uma instância de DefinirDiaUseCase com o repositório injetado.
func NovoDefinirDiaUseCase(repo repositorioDateException) *DefinirDiaUseCase {
	return &DefinirDiaUseCase{repo: repo}
}

// Executar valida os blocos (quando o tipo é extra), monta a definição da data
// e a persiste, substituindo uma definição anterior da mesma data se existir.
func (uc *DefinirDiaUseCase) Executar(in DefinirDiaInput) (*DefinirDiaOutput, error) {
	// Reaproveita o ID de uma definição existente para o upsert manter a
	// mesma linha (e os blocos antigos serem substituídos, não somados).
	id := uuid.NewString()
	existente, err := uc.repo.BuscarPorData(in.ProviderID, in.Data)
	if err != nil {
		return nil, err
	}
	if existente != nil {
		id = existente.ID
	}

	var blocos []availability.TimeBlock
	for _, b := range in.Blocos {
		bloco, err := availability.NovoTimeBlock(b.InicioMinutos, b.FimMinutos)
		if err != nil {
			return nil, err
		}
		blocos = append(blocos, bloco)
	}

	excecao, err := availability.NovaDateException(id, in.ProviderID, in.Data, in.Tipo, blocos)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.SalvarExcecao(excecao); err != nil {
		return nil, err
	}

	return &DefinirDiaOutput{
		Data:   excecao.Data,
		Tipo:   excecao.Tipo,
		Blocos: excecao.Blocos,
	}, nil
}
