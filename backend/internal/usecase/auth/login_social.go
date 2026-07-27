package auth

import (
	"context"
	"errors"
	"time"

	"agendago/internal/domain/client"
	"agendago/internal/domain/oauthstate"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/session"
	"agendago/internal/domain/socialidentity"
	"agendago/internal/pkg/token"

	"github.com/google/uuid"
)

// ErrStateInvalido é retornado quando o state do callback OAuth não confere
// com nenhum state emitido (expirado, já consumido, ou forjado) — sinal de
// possível CSRF.
var ErrStateInvalido = errors.New("state inválido ou expirado")

// ErrEmailNaoVerificado é retornado quando o provedor social não confirma que
// o email do usuário foi verificado — vincular por email ou criar conta nessa
// condição abriria brecha para account takeover.
var ErrEmailNaoVerificado = errors.New("email não verificado pelo provedor")

// ErrEmailJaCadastradoOutroTipo é retornado quando o email já pertence a um
// usuário do outro tipo (prestador tentando entrar como cliente, ou
// vice-versa) — um email só pode existir em um dos dois papéis, mesma regra
// do cadastro por senha.
var ErrEmailJaCadastradoOutroTipo = errors.New("email já cadastrado como outro tipo de conta")

// ErrContaNaoEncontrada é retornado quando alguém entra pelo login social sem
// ter conta. Só acontece no fluxo de PublicoLogin: ali não há tipo declarado,
// e criar a conta exigiria adivinhar se a pessoa é cliente ou prestador — uma
// escolha que muda o que ela pode fazer no sistema e que não cabe ao backend
// chutar. O frontend manda para o cadastro, onde o tipo é escolhido.
var ErrContaNaoEncontrada = errors.New("não há conta para este email")

// TTLOAuthState é a validade do state emitido antes do redirect ao provedor.
const TTLOAuthState = 10 * time.Minute

// TelefonePendente preenche o telefone (exigido pelo domínio) de um
// prestador criado via login social — o provedor OIDC não fornece telefone.
// A agenda nasce desativada (AceitaAgendamentos=false), então não há risco de
// receber agendamentos antes do prestador completar o telefone real em
// Preferências (DefinirTelefone, que exige um valor válido de verdade).
// Exportado (não interno) porque PerfilOutput usa para sinalizar ao frontend
// que o telefone ainda não foi confirmado — ver perfil.go.
const TelefonePendente = "00000000"

// provedorOIDC troca um código de autorização por uma identidade OIDC
// verificada. Implementado pelos adapters em adapter/oauth (ex.: Google).
type provedorOIDC interface {
	URLAutorizacao(state, nonce string) string
	TrocarCodigo(ctx context.Context, code, nonceEsperado string) (*socialidentity.IdentidadeOIDC, error)
}

// repositorioIdentidadeSocial vincula um (provedor, sub) a um usuário existente.
type repositorioIdentidadeSocial interface {
	Salvar(i *socialidentity.Identidade) error
	BuscarPorProvedorSub(provedor socialidentity.Provedor, sub string) (*socialidentity.Identidade, error)
	RemoverDoUsuario(userID string) error
}

// repositorioOAuthState persiste o state de uso único do fluxo OAuth (CSRF).
type repositorioOAuthState interface {
	Salvar(s *oauthstate.State) error
	Consumir(stateHash string) (*oauthstate.State, error)
	RemoverExpirados() error
}

// criadorClient cria e persiste um novo cliente sem senha (login social).
type criadorClient interface {
	Salvar(c *client.Client) error
}

// criadorProvider cria e persiste um novo prestador sem senha (login social).
type criadorProvider interface {
	Salvar(p *provider.Provider) error
}

// PublicoLoginSocial identifica com que intenção o fluxo social começou.
type PublicoLoginSocial string

const (
	// PublicoClient indica que o login social se aplica a um cliente.
	PublicoClient PublicoLoginSocial = "client"
	// PublicoProvider indica que o login social se aplica a um prestador.
	PublicoProvider PublicoLoginSocial = "provider"
	// PublicoLogin é a entrada de quem já tem conta e não declara o tipo: o
	// sistema descobre sozinho, pelo vínculo social ou pelo email. Usado na
	// tela de login, onde perguntar "cliente ou prestador?" é redundante —
	// a conta já existe e só pode ser de um dos dois.
	PublicoLogin PublicoLoginSocial = "login"
)

// LoginSocialUseCase autentica um cliente ou prestador via provedor OIDC
// (login social), criando a conta na primeira vez que o email aparece e
// vinculando-a nas vezes seguintes.
type LoginSocialUseCase struct {
	google       provedorOIDC
	clients      contaClient
	providers    contaProvider
	criaClient   criadorClient
	criaProvider criadorProvider
	identidades  repositorioIdentidadeSocial
	states       repositorioOAuthState
	sessoes      repositorioSessao
	hasher       hasherSenha
}

// NovoLoginSocialUseCase cria uma instância de LoginSocialUseCase com as
// dependências injetadas.
func NovoLoginSocialUseCase(
	google provedorOIDC,
	clients contaClient,
	providers contaProvider,
	criaClient criadorClient,
	criaProvider criadorProvider,
	identidades repositorioIdentidadeSocial,
	states repositorioOAuthState,
	sessoes repositorioSessao,
	hasher hasherSenha,
) *LoginSocialUseCase {
	return &LoginSocialUseCase{
		google:       google,
		clients:      clients,
		providers:    providers,
		criaClient:   criaClient,
		criaProvider: criaProvider,
		identidades:  identidades,
		states:       states,
		sessoes:      sessoes,
		hasher:       hasher,
	}
}

// Iniciar gera o state e o nonce do fluxo, persiste o state com o publico
// (client ou provider) já gravado nele, e devolve a URL de consentimento do
// Google, o state em texto puro (para o cookie curto do navegador) e o nonce
// (para validar o id_token no callback via cookie próprio, já que ele não é
// persistido). publico fica só no registro server-side do state — não é
// devolvido para virar cookie, para Concluir nunca precisar confiar num
// cookie separado sem vínculo criptográfico com o state consumido.
func (uc *LoginSocialUseCase) Iniciar(publico PublicoLoginSocial) (urlAutorizacao, stateTexto, nonce string, err error) {
	stateTexto, err = token.Gerar()
	if err != nil {
		return "", "", "", err
	}
	nonce, err = token.Gerar()
	if err != nil {
		return "", "", "", err
	}

	s := oauthstate.Novo(token.Hash(stateTexto), string(socialidentity.Google), string(publico), TTLOAuthState)
	if err := uc.states.Salvar(s); err != nil {
		return "", "", "", err
	}
	uc.states.RemoverExpirados()

	return uc.google.URLAutorizacao(stateTexto, nonce), stateTexto, nonce, nil
}

// Concluir valida o state (CSRF), troca o código pela identidade OIDC
// verificada e resolve o usuário: identidade já vinculada loga direto; email
// já existente e verificado vincula a identidade a esse usuário (preservando
// ID e histórico) e loga; email inédito cria uma conta nova sem senha —
// prestador nasce com TelefonePendente, que ele completa em Preferências. O
// tipo de conta (client/provider) vem do state consumido — verificado
// server-side — nunca de um cookie ou parâmetro externo. Retorna
// ErrStateInvalido em CSRF ou state sem publico reconhecido,
// ErrEmailNaoVerificado quando o provedor não confirma o email,
// ErrEmailJaCadastradoOutroTipo quando o email já é conta do outro tipo.
func (uc *LoginSocialUseCase) Concluir(ctx context.Context, code, stateRecebido, stateCookie, nonce string) (*LoginOutput, error) {
	if stateRecebido == "" || stateCookie == "" || stateRecebido != stateCookie {
		return nil, ErrStateInvalido
	}

	guardado, err := uc.states.Consumir(token.Hash(stateRecebido))
	if err != nil {
		return nil, err
	}
	if guardado == nil || guardado.Expirado(time.Now()) {
		return nil, ErrStateInvalido
	}

	publico := PublicoLoginSocial(guardado.Publico)
	if publico != PublicoClient && publico != PublicoProvider && publico != PublicoLogin {
		return nil, ErrStateInvalido
	}

	identidadeOIDC, err := uc.google.TrocarCodigo(ctx, code, nonce)
	if err != nil {
		return nil, err
	}

	// Identidade já vinculada: o tipo vem do vínculo, não do que a URL pediu.
	// É o registro no banco que sabe se aquela conta Google é de cliente ou de
	// prestador — confiar no parâmetro faria "Sou prestador" tentar abrir uma
	// sessão de prestador para um usuário que é cliente.
	vinculo, err := uc.identidades.BuscarPorProvedorSub(socialidentity.Google, identidadeOIDC.Sub)
	if err != nil {
		return nil, err
	}
	if vinculo != nil {
		return uc.criarSessaoParaUsuarioExistente(vinculo.UserID, session.TipoUsuario(vinculo.UserType))
	}

	// Sem vínculo e sem tipo declarado: procura a conta pelo email. Só entra
	// quem já existe — criar exigiria adivinhar o tipo (ver ErrContaNaoEncontrada).
	if publico == PublicoLogin {
		return uc.resolverPorEmail(identidadeOIDC)
	}

	if publico == PublicoProvider {
		return uc.resolverProvider(identidadeOIDC)
	}
	return uc.resolverClient(identidadeOIDC)
}

// resolverPorEmail descobre o tipo da conta a partir do email verificado pelo
// provedor. Reusa resolverProvider/resolverClient, que já vinculam a
// identidade social e tratam o convidado (cliente sem senha) virando conta.
func (uc *LoginSocialUseCase) resolverPorEmail(id *socialidentity.IdentidadeOIDC) (*LoginOutput, error) {
	// Antes de consultar qualquer tabela: um email não verificado poderia
	// apontar para o endereço de outra pessoa, e aqui ele é a única prova de
	// identidade que temos.
	if !id.EmailVerificado {
		return nil, ErrEmailNaoVerificado
	}

	prestador, err := uc.providers.BuscarPorEmail(id.Email)
	if err != nil {
		return nil, err
	}
	if prestador != nil {
		return uc.resolverProvider(id)
	}

	cliente, err := uc.clients.BuscarPorEmail(id.Email)
	if err != nil {
		return nil, err
	}
	if cliente != nil {
		return uc.resolverClient(id)
	}

	return nil, ErrContaNaoEncontrada
}

// resolverClient resolve o usuário cliente para o email do provedor social.
// A verificação de EmailVerificado vem antes de qualquer consulta: sem ela,
// nem vincular a uma conta existente nem criar uma nova são seguros — um
// provedor que devolvesse um email não verificado poderia, em tese, apontar
// para o endereço de outra pessoa.
func (uc *LoginSocialUseCase) resolverClient(id *socialidentity.IdentidadeOIDC) (*LoginOutput, error) {
	if !id.EmailVerificado {
		return nil, ErrEmailNaoVerificado
	}

	// o email não pode já pertencer a um prestador — mesma regra do
	// cadastro por senha (ver usecase/provider/cadastrar.go): um email só
	// existe em um dos dois papéis no sistema.
	prestadorExistente, err := uc.providers.BuscarPorEmail(id.Email)
	if err != nil {
		return nil, err
	}
	if prestadorExistente != nil {
		return nil, ErrEmailJaCadastradoOutroTipo
	}

	existente, err := uc.clients.BuscarPorEmail(id.Email)
	if err != nil {
		return nil, err
	}

	var c *client.Client
	if existente != nil {
		c = existente
	} else {
		senhaHash, err := uc.senhaSentinela()
		if err != nil {
			return nil, err
		}
		c, err = client.NovoComConta(uuid.NewString(), id.Nome, id.Email, senhaHash)
		if err != nil {
			return nil, err
		}
		if err := uc.criaClient.Salvar(c); err != nil {
			return nil, err
		}
	}
	if !c.Ativo {
		return nil, ErrUsuarioInativo
	}

	if err := uc.vincularIdentidade(id, c.ID, session.TipoClient); err != nil {
		return nil, err
	}
	return uc.novaSessao(c.ID, c.Nome, session.TipoClient)
}

// resolverProvider resolve o usuário prestador para o email do provedor
// social. Mesma ordem de verificação de resolverClient: email verificado
// primeiro, depois a checagem cross-type, só então busca/cria a conta. Um
// prestador novo nasce com TelefonePendente — o frontend detecta isso (via
// PerfilOutput.TelefonePendente) e força a ida a Preferências antes de
// liberar o resto do painel.
func (uc *LoginSocialUseCase) resolverProvider(id *socialidentity.IdentidadeOIDC) (*LoginOutput, error) {
	if !id.EmailVerificado {
		return nil, ErrEmailNaoVerificado
	}

	// o email não pode já pertencer a um cliente — mesma regra do cadastro
	// por senha (ver usecase/provider/cadastrar.go).
	clienteExistente, err := uc.clients.BuscarPorEmail(id.Email)
	if err != nil {
		return nil, err
	}
	if clienteExistente != nil {
		return nil, ErrEmailJaCadastradoOutroTipo
	}

	existente, err := uc.providers.BuscarPorEmail(id.Email)
	if err != nil {
		return nil, err
	}

	var p *provider.Provider
	if existente != nil {
		p = existente
	} else {
		senhaHash, err := uc.senhaSentinela()
		if err != nil {
			return nil, err
		}
		p, err = provider.Novo(uuid.NewString(), id.Nome, id.Email, TelefonePendente, senhaHash)
		if err != nil {
			return nil, err
		}
		if err := uc.criaProvider.Salvar(p); err != nil {
			return nil, err
		}
	}
	if !p.Ativo {
		return nil, ErrUsuarioInativo
	}

	if err := uc.vincularIdentidade(id, p.ID, session.TipoProvider); err != nil {
		return nil, err
	}
	return uc.novaSessao(p.ID, p.Nome, session.TipoProvider)
}

func (uc *LoginSocialUseCase) vincularIdentidade(id *socialidentity.IdentidadeOIDC, userID string, userType session.TipoUsuario) error {
	vinculo := socialidentity.Nova(uuid.NewString(), socialidentity.Google, id.Sub, userID, string(userType), id.Email)
	return uc.identidades.Salvar(vinculo)
}

func (uc *LoginSocialUseCase) criarSessaoParaUsuarioExistente(userID string, userType session.TipoUsuario) (*LoginOutput, error) {
	var nome string
	var ativo bool
	if userType == session.TipoProvider {
		p, err := uc.providers.BuscarPorID(userID)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, ErrCredenciaisInvalidas
		}
		nome, ativo = p.Nome, p.Ativo
	} else {
		c, err := uc.clients.BuscarPorID(userID)
		if err != nil {
			return nil, err
		}
		if c == nil {
			return nil, ErrCredenciaisInvalidas
		}
		nome, ativo = c.Nome, c.Ativo
	}
	if !ativo {
		return nil, ErrUsuarioInativo
	}
	return uc.novaSessao(userID, nome, userType)
}

func (uc *LoginSocialUseCase) novaSessao(userID, nome string, userType session.TipoUsuario) (*LoginOutput, error) {
	t, err := token.Gerar()
	if err != nil {
		return nil, err
	}

	s := session.Nova(token.Hash(t), userID, userType, TTLSessao)
	if err := uc.sessoes.Salvar(s); err != nil {
		return nil, err
	}
	uc.sessoes.RemoverExpiradas()

	return &LoginOutput{
		Token:    t,
		ExpiraEm: s.ExpiraEm,
		UserID:   userID,
		Nome:     nome,
	}, nil
}

// senhaSentinela gera um hash de senha aleatória de 256 bits, nunca
// comunicada ao usuário — só existe para satisfazer a invariante de domínio
// de que toda conta com login (TemConta/Novo) tem um SenhaHash não vazio.
// Quem loga via provedor social nunca autentica por essa senha.
func (uc *LoginSocialUseCase) senhaSentinela() (string, error) {
	aleatoria, err := token.Gerar()
	if err != nil {
		return "", err
	}
	return uc.hasher.Gerar(aleatoria)
}
