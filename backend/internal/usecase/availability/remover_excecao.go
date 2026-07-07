package availability

// RemoverExcecaoInput identifica a exceção a remover e o prestador dono da sessão.
type RemoverExcecaoInput struct {
	ProviderID string
	ExcecaoID  string
}

// RemoverExcecaoUseCase remove uma exceção de data do prestador.
type RemoverExcecaoUseCase struct {
	repo repositorioDateException
}

// NovoRemoverExcecaoUseCase cria uma instância de RemoverExcecaoUseCase com o repositório injetado.
func NovoRemoverExcecaoUseCase(repo repositorioDateException) *RemoverExcecaoUseCase {
	return &RemoverExcecaoUseCase{repo: repo}
}

// Executar remove a exceção informada. Retorna ErrExcecaoNaoEncontrada tanto
// quando a exceção não existe quanto quando pertence a outro prestador — não
// distingue os dois casos para não vazar a existência de recursos de terceiros.
func (uc *RemoverExcecaoUseCase) Executar(in RemoverExcecaoInput) error {
	excecao, err := uc.repo.BuscarPorID(in.ExcecaoID)
	if err != nil {
		return err
	}
	if excecao == nil || excecao.ProviderID != in.ProviderID {
		return ErrExcecaoNaoEncontrada
	}

	return uc.repo.Remover(in.ExcecaoID)
}
