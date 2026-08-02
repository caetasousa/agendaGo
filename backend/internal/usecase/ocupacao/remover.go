package ocupacao

// RemoverInput identifica o compromisso e quem está removendo. ProviderID vem
// da sessão, nunca do corpo da requisição.
type RemoverInput struct {
	ID         string
	ProviderID string
}

// RemoverUseCase apaga um compromisso pessoal, devolvendo o horário à oferta.
type RemoverUseCase struct {
	ocupacoes repositorioOcupacao
}

// NovoRemoverUseCase cria uma instância de RemoverUseCase com o repositório injetado.
func NovoRemoverUseCase(ocupacoes repositorioOcupacao) *RemoverUseCase {
	return &RemoverUseCase{ocupacoes: ocupacoes}
}

// Executar remove o compromisso, desde que ele pertença à agenda de quem pediu.
//
// Retorna ErrOcupacaoNaoEncontrada tanto quando o id não existe quanto quando
// pertence a outra agenda — distinguir os dois casos contaria a um prestador
// que certo id existe na agenda de outro.
func (uc *RemoverUseCase) Executar(in RemoverInput) error {
	o, err := uc.ocupacoes.BuscarPorID(in.ID)
	if err != nil {
		return err
	}
	if o == nil || o.ProviderID != in.ProviderID {
		return ErrOcupacaoNaoEncontrada
	}
	return uc.ocupacoes.Remover(in.ID)
}
