package membro

import (
	"time"

	"agendago/internal/domain/membro"
	"agendago/internal/domain/usuario"
	"agendago/internal/pkg/token"
)

// ConviteOutput descreve um convite para a tela de aceite: de qual agenda ele
// é e para qual email foi emitido. O email volta para preencher o formulário,
// já que quem aceita não deve poder trocá-lo.
type ConviteOutput struct {
	Email      string
	NomeAgenda string
	Papel      string
}

// ConsultarConviteUseCase lê um convite sem consumi-lo, para a tela mostrar de
// quem é a agenda antes de a pessoa decidir aceitar.
type ConsultarConviteUseCase struct {
	convites  repositorioConvite
	providers buscadorProvider
}

// NovoConsultarConviteUseCase cria uma instância de ConsultarConviteUseCase.
func NovoConsultarConviteUseCase(convites repositorioConvite, providers buscadorProvider) *ConsultarConviteUseCase {
	return &ConsultarConviteUseCase{convites: convites, providers: providers}
}

// Executar resolve o convite do token. Retorna ErrConviteInvalido tanto para
// token inexistente quanto para expirado. Consultar não gasta o convite: a
// pessoa pode abrir o link mais de uma vez antes de preencher o formulário.
func (uc *ConsultarConviteUseCase) Executar(tokenPuro string) (*ConviteOutput, error) {
	c, err := uc.convites.BuscarPorTokenHash(token.Hash(tokenPuro))
	if err != nil {
		return nil, err
	}
	if c == nil || c.Expirado(time.Now()) {
		return nil, ErrConviteInvalido
	}

	p, err := uc.providers.BuscarPorID(c.ProviderID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrConviteInvalido
	}

	return &ConviteOutput{Email: c.Email, NomeAgenda: p.Nome, Papel: string(c.Papel)}, nil
}

// AceitarInput traz o token e os dados que quem aceita preenche. O email não
// vem daqui: ele é o do convite, para ninguém redirecionar um convite alheio.
type AceitarInput struct {
	Token    string
	Telefone string
	Senha    string
}

// AceitarOutput identifica a conta criada e a agenda a que ela ganhou acesso.
type AceitarOutput struct {
	UsuarioID  string
	ProviderID string
	NomeAgenda string
}

// AceitarConviteUseCase conclui o convite: cria a conta de quem foi convidado e
// o vínculo com a agenda.
type AceitarConviteUseCase struct {
	convites  repositorioConvite
	usuarios  repositorioUsuario
	membros   repositorioMembro
	providers buscadorProvider
	hasher    hasherSenha
}

// NovoAceitarConviteUseCase cria uma instância de AceitarConviteUseCase com as dependências injetadas.
func NovoAceitarConviteUseCase(
	convites repositorioConvite,
	usuarios repositorioUsuario,
	membros repositorioMembro,
	providers buscadorProvider,
	hasher hasherSenha,
) *AceitarConviteUseCase {
	return &AceitarConviteUseCase{convites: convites, usuarios: usuarios, membros: membros, providers: providers, hasher: hasher}
}

// Executar consome o convite (uso único) e cria a conta com um único vínculo:
// o da agenda que convidou. Nenhuma agenda nova é criada — é isso que
// diferencia entrar por convite de se cadastrar como prestador, e é o que
// mantém a resolução da agenda determinística para essa pessoa.
//
// Retorna ErrConviteInvalido para token inexistente, expirado, ou quando o
// email deixou de estar livre entre o convite e o aceite.
func (uc *AceitarConviteUseCase) Executar(in AceitarInput) (*AceitarOutput, error) {
	c, err := uc.convites.Consumir(token.Hash(in.Token))
	if err != nil {
		return nil, err
	}
	if c == nil || c.Expirado(time.Now()) {
		return nil, ErrConviteInvalido
	}

	p, err := uc.providers.BuscarPorID(c.ProviderID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrConviteInvalido
	}

	// O email pode ter sido registrado entre o envio e o aceite — por um
	// cadastro comum, por exemplo. Sem esta checagem o INSERT falharia na
	// constraint de unicidade, com um erro que não diz nada a quem aceita.
	if existente, err := uc.usuarios.BuscarPorEmail(c.Email); err != nil {
		return nil, err
	} else if existente != nil {
		return nil, ErrConviteInvalido
	}

	senhaHash, err := uc.hasher.Gerar(in.Senha)
	if err != nil {
		return nil, err
	}

	u, err := usuario.Novo(idNovo(), c.Email, in.Telefone, senhaHash)
	if err != nil {
		return nil, err
	}
	if err := uc.usuarios.Salvar(u); err != nil {
		return nil, err
	}

	vinculo, err := membro.Novo(idNovo(), u.ID, c.ProviderID, c.Papel)
	if err != nil {
		return nil, err
	}
	if err := uc.membros.Salvar(vinculo); err != nil {
		return nil, err
	}

	return &AceitarOutput{UsuarioID: u.ID, ProviderID: c.ProviderID, NomeAgenda: p.Nome}, nil
}
