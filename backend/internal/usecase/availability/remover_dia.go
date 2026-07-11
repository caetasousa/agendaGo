package availability

import "time"

// RemoverDiaInput identifica a data cuja definição própria será removida e o
// prestador dono da sessão.
type RemoverDiaInput struct {
	ProviderID string
	Data       time.Time
}

// RemoverDiaUseCase remove a definição própria de uma data, fazendo o dia
// voltar ao expediente padrão do prestador.
type RemoverDiaUseCase struct {
	repo repositorioDateException
}

// NovoRemoverDiaUseCase cria uma instância de RemoverDiaUseCase com o repositório injetado.
func NovoRemoverDiaUseCase(repo repositorioDateException) *RemoverDiaUseCase {
	return &RemoverDiaUseCase{repo: repo}
}

// Executar remove a definição da data informada. Retorna ErrDiaNaoDefinido
// quando a data não tem definição própria deste prestador.
func (uc *RemoverDiaUseCase) Executar(in RemoverDiaInput) error {
	excecao, err := uc.repo.BuscarPorData(in.ProviderID, in.Data)
	if err != nil {
		return err
	}
	if excecao == nil {
		return ErrDiaNaoDefinido
	}

	return uc.repo.Remover(excecao.ID)
}
