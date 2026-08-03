package provider

import (
	"errors"

	"agendago/internal/domain/availability"
	"agendago/internal/domain/membro"
)

// ErrProviderNaoEncontrado é retornado quando o prestador da sessão não existe mais.
var ErrProviderNaoEncontrado = errors.New("prestador não encontrado")

// ErrEquipeComMembros é retornado ao desligar o recurso de equipe numa agenda
// que ainda tem alguém além do dono — ou um convite no ar. Desligar assim
// esconderia da dona gente que continua com acesso: primeiro remove o acesso,
// depois desliga.
var ErrEquipeComMembros = errors.New("remova quem ainda tem acesso antes de desativar a equipe")

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
	PermiteEquipe                bool
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
	PermiteEquipe                bool
}

// AtualizarPreferenciasUseCase orquestra a atualização das preferências de um prestador.
type AtualizarPreferenciasUseCase struct {
	repo     repositorioPreferencias
	usuarios repositorioUsuario
	membros  listadorMembros
	convites listadorConvitesPendentes
}

// NovoAtualizarPreferenciasUseCase cria uma instância de AtualizarPreferenciasUseCase com os repositórios injetados.
func NovoAtualizarPreferenciasUseCase(
	repo repositorioPreferencias,
	usuarios repositorioUsuario,
	membros listadorMembros,
	convites listadorConvitesPendentes,
) *AtualizarPreferenciasUseCase {
	return &AtualizarPreferenciasUseCase{repo: repo, usuarios: usuarios, membros: membros, convites: convites}
}

// Executar carrega a conta e a agenda, aplica as preferências via regras de
// domínio e persiste as duas. Retorna ErrProviderNaoEncontrado se qualquer uma
// das duas não existir, ErrDescansoInvalido se o descanso for negativo e
// ErrEquipeComMembros ao desligar a equipe com alguém ainda dentro.
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

	if in.PermiteEquipe {
		p.AtivarEquipe()
	} else {
		// A checagem só faz sentido no desligamento de fato: com o recurso já
		// desligado não há vínculo nem convite a proteger.
		if p.PermiteEquipe {
			acompanhada, err := uc.agendaTemMaisAlguem(in.ProviderID)
			if err != nil {
				return nil, err
			}
			if acompanhada {
				return nil, ErrEquipeComMembros
			}
		}
		p.DesativarEquipe()
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
		PermiteEquipe:                p.PermiteEquipe,
	}, nil
}

// agendaTemMaisAlguem diz se alguém além do dono opera a agenda — ou está a um
// clique de operar, com convite pendente.
func (uc *AtualizarPreferenciasUseCase) agendaTemMaisAlguem(providerID string) (bool, error) {
	vinculos, err := uc.membros.ListarPorProvider(providerID)
	if err != nil {
		return false, err
	}
	for _, v := range vinculos {
		if v.Papel != membro.PapelDono {
			return true, nil
		}
	}

	pendentes, err := uc.convites.ListarPendentesPorProvider(providerID)
	if err != nil {
		return false, err
	}
	return len(pendentes) > 0, nil
}
