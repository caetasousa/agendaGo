package provider

import (
	"errors"

	"agendago/internal/domain/availability"
)

// ErrProviderNaoEncontrado é retornado quando o prestador da sessão não existe mais.
var ErrProviderNaoEncontrado = errors.New("prestador não encontrado")

// BlocoInput representa um bloco do expediente padrão, ainda não validado pelo domínio.
type BlocoInput struct {
	InicioMinutos int
	FimMinutos    int
}

// AtualizarPreferenciasInput contém as preferências a aplicar. ProviderID vem
// da identidade da sessão autenticada, nunca do corpo da requisição.
type AtualizarPreferenciasInput struct {
	ProviderID         string
	AceitaAgendamentos bool
	DescansoMinutos    int
	HorariosPadrao     []BlocoInput
}

// AtualizarPreferenciasOutput contém as preferências após a atualização.
type AtualizarPreferenciasOutput struct {
	AceitaAgendamentos bool
	DescansoMinutos    int
	HorariosPadrao     []availability.TimeBlock
}

// AtualizarPreferenciasUseCase orquestra a atualização das preferências de um prestador.
type AtualizarPreferenciasUseCase struct {
	repo repositorioPreferencias
}

// NovoAtualizarPreferenciasUseCase cria uma instância de AtualizarPreferenciasUseCase com o repositório injetado.
func NovoAtualizarPreferenciasUseCase(repo repositorioPreferencias) *AtualizarPreferenciasUseCase {
	return &AtualizarPreferenciasUseCase{repo: repo}
}

// Executar carrega o prestador, aplica as preferências via regras de domínio
// e persiste. Retorna ErrProviderNaoEncontrado se o prestador não existir e
// ErrDescansoInvalido se o descanso for negativo.
func (uc *AtualizarPreferenciasUseCase) Executar(in AtualizarPreferenciasInput) (*AtualizarPreferenciasOutput, error) {
	p, err := uc.repo.BuscarPorID(in.ProviderID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProviderNaoEncontrado
	}

	if in.AceitaAgendamentos {
		p.AtivarAgenda()
	} else {
		p.DesativarAgenda()
	}

	if err := p.DefinirDescanso(in.DescansoMinutos); err != nil {
		return nil, err
	}

	blocos := make([]availability.TimeBlock, 0, len(in.HorariosPadrao))
	for _, b := range in.HorariosPadrao {
		bloco, err := availability.NovoTimeBlock(b.InicioMinutos, b.FimMinutos)
		if err != nil {
			return nil, err
		}
		blocos = append(blocos, bloco)
	}
	if err := p.DefinirHorariosPadrao(blocos); err != nil {
		return nil, err
	}

	if err := uc.repo.Atualizar(p); err != nil {
		return nil, err
	}

	return &AtualizarPreferenciasOutput{
		AceitaAgendamentos: p.AceitaAgendamentos,
		DescansoMinutos:    p.DescansoMinutos,
		HorariosPadrao:     p.HorariosPadrao,
	}, nil
}
