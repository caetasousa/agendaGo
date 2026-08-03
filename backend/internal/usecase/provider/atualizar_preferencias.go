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

// AtualizarPreferenciasInput contém as preferências a aplicar. UsuarioID e
// ProviderID vêm da identidade da sessão autenticada, nunca do corpo da
// requisição. São dois porque a tela de Configurações mexe nas duas coisas de
// uma vez: o telefone é da conta, o resto é da agenda.
type AtualizarPreferenciasInput struct {
	UsuarioID                    string
	ProviderID                   string
	Telefone                     string
	AceitaAgendamentos           bool
	DescansoMinutos              int
	DuracaoAtendimentoMinutos    int
	HorariosPadrao               []BlocoInput
	PermiteMarcacaoPeloPrestador bool
	// Slug vazio mantém o endereço atual. O prestador só o troca quando quer,
	// e trocar quebra os links já compartilhados — ver DefinirSlug.
	Slug string
}

// AtualizarPreferenciasOutput contém as preferências após a atualização.
type AtualizarPreferenciasOutput struct {
	Slug                         string
	Telefone                     string
	AceitaAgendamentos           bool
	DescansoMinutos              int
	DuracaoAtendimentoMinutos    int
	HorariosPadrao               []availability.TimeBlock
	PermiteMarcacaoPeloPrestador bool
}

// AtualizarPreferenciasUseCase orquestra a atualização das preferências de um prestador.
type AtualizarPreferenciasUseCase struct {
	repo     repositorioPreferencias
	usuarios repositorioUsuario
}

// NovoAtualizarPreferenciasUseCase cria uma instância de AtualizarPreferenciasUseCase com os repositórios injetados.
func NovoAtualizarPreferenciasUseCase(repo repositorioPreferencias, usuarios repositorioUsuario) *AtualizarPreferenciasUseCase {
	return &AtualizarPreferenciasUseCase{repo: repo, usuarios: usuarios}
}

// Executar carrega a conta e a agenda, aplica as preferências via regras de
// domínio e persiste as duas. Retorna ErrProviderNaoEncontrado se qualquer uma
// das duas não existir e ErrDescansoInvalido se o descanso for negativo.
func (uc *AtualizarPreferenciasUseCase) Executar(in AtualizarPreferenciasInput) (*AtualizarPreferenciasOutput, error) {
	p, err := uc.repo.BuscarPorID(in.ProviderID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProviderNaoEncontrado
	}

	u, err := uc.usuarios.BuscarPorID(in.UsuarioID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrProviderNaoEncontrado
	}

	// O telefone é da conta de quem está logado, não da agenda: um operador
	// que edite as preferências troca o próprio telefone, não o do dono.
	if err := u.DefinirTelefone(in.Telefone); err != nil {
		return nil, err
	}

	if in.AceitaAgendamentos {
		p.AtivarAgenda()
	} else {
		p.DesativarAgenda()
	}

	if in.PermiteMarcacaoPeloPrestador {
		p.AtivarMarcacaoPeloPrestador()
	} else {
		p.DesativarMarcacaoPeloPrestador()
	}

	if err := p.DefinirDescanso(in.DescansoMinutos); err != nil {
		return nil, err
	}

	if err := p.DefinirDuracaoAtendimento(in.DuracaoAtendimentoMinutos); err != nil {
		return nil, err
	}

	if in.Slug != "" && in.Slug != p.Slug {
		if err := p.DefinirSlug(in.Slug); err != nil {
			return nil, err
		}
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
	if err := uc.usuarios.Atualizar(u); err != nil {
		return nil, err
	}

	return &AtualizarPreferenciasOutput{
		Slug:                         p.Slug,
		Telefone:                     u.Telefone,
		AceitaAgendamentos:           p.AceitaAgendamentos,
		DescansoMinutos:              p.DescansoMinutos,
		DuracaoAtendimentoMinutos:    p.DuracaoAtendimentoMinutos,
		HorariosPadrao:               p.HorariosPadrao,
		PermiteMarcacaoPeloPrestador: p.PermiteMarcacaoPeloPrestador,
	}, nil
}
