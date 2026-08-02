package ocupacao

import (
	"time"

	domocupacao "agendago/internal/domain/ocupacao"
)

// ListarInput define o prestador e o período (inclusivo) da consulta.
type ListarInput struct {
	ProviderID string
	De         time.Time
	Ate        time.Time
}

// ListarUseCase lista os compromissos pessoais de um período.
type ListarUseCase struct {
	ocupacoes repositorioOcupacao
}

// NovoListarUseCase cria uma instância de ListarUseCase com o repositório injetado.
func NovoListarUseCase(ocupacoes repositorioOcupacao) *ListarUseCase {
	return &ListarUseCase{ocupacoes: ocupacoes}
}

// Executar devolve os compromissos do período, ordenados por data e horário.
// Retorna ErrPeriodoInvalido para período invertido ou maior que maxDiasPeriodo.
func (uc *ListarUseCase) Executar(in ListarInput) ([]*domocupacao.Ocupacao, error) {
	if in.Ate.Before(in.De) || in.Ate.Sub(in.De) > maxDiasPeriodo*24*time.Hour {
		return nil, ErrPeriodoInvalido
	}
	return uc.ocupacoes.ListarPorPeriodo(in.ProviderID, in.De, in.Ate)
}
