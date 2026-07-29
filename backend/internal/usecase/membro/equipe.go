package membro

import (
	"time"

	"agendago/internal/domain/membro"
)

// MembroResumo descreve quem tem acesso a uma agenda, na tela de equipe.
type MembroResumo struct {
	ID       string
	Email    string
	Papel    string
	Ativo    bool
	EhDono   bool
	CriadoEm time.Time
}

// ConvitePendente descreve um convite emitido e ainda não aceito.
type ConvitePendente struct {
	Email    string
	Papel    string
	ExpiraEm time.Time
}

// EquipeOutput reúne quem já entrou e quem ainda foi só convidado.
type EquipeOutput struct {
	Membros   []MembroResumo
	Pendentes []ConvitePendente
}

// ListarEquipeUseCase monta a visão de quem opera uma agenda.
type ListarEquipeUseCase struct {
	membros  repositorioMembro
	usuarios repositorioUsuario
	convites repositorioConvite
}

// NovoListarEquipeUseCase cria uma instância de ListarEquipeUseCase com as dependências injetadas.
func NovoListarEquipeUseCase(membros repositorioMembro, usuarios repositorioUsuario, convites repositorioConvite) *ListarEquipeUseCase {
	return &ListarEquipeUseCase{membros: membros, usuarios: usuarios, convites: convites}
}

// Executar devolve os vínculos da agenda com o email de cada pessoa, mais os
// convites ainda pendentes. Uma consulta por vínculo para resolver a conta: a
// equipe de uma agenda é pequena, e evita espalhar um join de leitura pelo
// repositório de vínculos.
func (uc *ListarEquipeUseCase) Executar(providerID string) (*EquipeOutput, error) {
	vinculos, err := uc.membros.ListarPorProvider(providerID)
	if err != nil {
		return nil, err
	}

	out := &EquipeOutput{Membros: make([]MembroResumo, 0, len(vinculos))}
	for _, v := range vinculos {
		resumo := MembroResumo{
			ID:       v.ID,
			Papel:    string(v.Papel),
			EhDono:   v.Papel == membro.PapelDono,
			CriadoEm: v.CriadoEm,
		}
		u, err := uc.usuarios.BuscarPorID(v.UsuarioID)
		if err != nil {
			return nil, err
		}
		// Vínculo sem conta não deveria existir; se existir, aparece na lista
		// sem email em vez de sumir dela — some da tela é pior de diagnosticar.
		if u != nil {
			resumo.Email = u.Email
			resumo.Ativo = u.Ativo
		}
		out.Membros = append(out.Membros, resumo)
	}

	pendentes, err := uc.convites.ListarPendentesPorProvider(providerID)
	if err != nil {
		return nil, err
	}
	out.Pendentes = make([]ConvitePendente, 0, len(pendentes))
	for _, c := range pendentes {
		out.Pendentes = append(out.Pendentes, ConvitePendente{
			Email: c.Email, Papel: string(c.Papel), ExpiraEm: c.ExpiraEm,
		})
	}
	return out, nil
}

// RemoverInput identifica o vínculo a remover. ProviderID vem da sessão: sem
// ele, o id de um vínculo alheio removeria acesso na agenda de outra pessoa.
type RemoverInput struct {
	ProviderID string
	MembroID   string
}

// RemoverMembroUseCase revoga o acesso de alguém a uma agenda.
type RemoverMembroUseCase struct {
	membros   repositorioMembro
	removedor removedorMembro
	usuarios  removedorUsuario
	sessoes   revogadorSessoes
}

// NovoRemoverMembroUseCase cria uma instância de RemoverMembroUseCase com as dependências injetadas.
func NovoRemoverMembroUseCase(
	membros repositorioMembro,
	removedor removedorMembro,
	usuarios removedorUsuario,
	sessoes revogadorSessoes,
) *RemoverMembroUseCase {
	return &RemoverMembroUseCase{membros: membros, removedor: removedor, usuarios: usuarios, sessoes: sessoes}
}

// Executar apaga o vínculo, derruba as sessões da pessoa removida e — se
// aquele era o único vínculo dela — apaga também a conta.
//
// Apagar a conta não é excesso: quem entrou por convite tem conta pela única
// razão de operar aquela agenda. Sem o vínculo ela não conseguiria mais entrar
// (o login exige um vínculo), e a conta órfã ainda ocuparia o email para
// sempre, impedindo que a pessoa fosse convidada de novo — o convite recusa
// qualquer email que já tenha conta.
//
// Quem tem outro vínculo mantém a conta: perdeu o acesso a ESTA agenda, não ao
// sistema.
//
// Retorna ErrMembroNaoEncontrado quando o vínculo não é desta agenda, e
// ErrNaoRemoveDono ao tentar remover o dono: uma agenda sem dono ficaria sem
// quem administre a conta.
func (uc *RemoverMembroUseCase) Executar(in RemoverInput) error {
	vinculos, err := uc.membros.ListarPorProvider(in.ProviderID)
	if err != nil {
		return err
	}

	for _, v := range vinculos {
		if v.ID != in.MembroID {
			continue
		}
		if v.Papel == membro.PapelDono {
			return ErrNaoRemoveDono
		}
		if err := uc.removedor.Remover(v.ID); err != nil {
			return err
		}
		if err := uc.sessoes.RemoverDoUsuario(v.UsuarioID); err != nil {
			return err
		}

		restante, err := uc.membros.BuscarPorUsuario(v.UsuarioID)
		if err != nil {
			return err
		}
		if restante == nil {
			return uc.usuarios.Remover(v.UsuarioID)
		}
		return nil
	}
	return ErrMembroNaoEncontrado
}
