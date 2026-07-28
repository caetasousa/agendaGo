package auth

import (
	"agendago/internal/domain/session"
	"agendago/internal/pkg/token"
)

// LoginProviderUseCase autentica um usuário do lado prestador e cria uma nova
// sessão. A credencial é do usuário; o nome exibido vem da agenda que ele
// opera.
type LoginProviderUseCase struct {
	usuarios  buscadorUsuario
	membros   buscadorMembro
	providers buscadorProvider
	sessoes   repositorioSessao
	hasher    hasherSenha
}

// NovoLoginProviderUseCase cria uma instância de LoginProviderUseCase com as dependências injetadas.
func NovoLoginProviderUseCase(
	usuarios buscadorUsuario,
	membros buscadorMembro,
	providers buscadorProvider,
	sessoes repositorioSessao,
	hasher hasherSenha,
) *LoginProviderUseCase {
	return &LoginProviderUseCase{usuarios: usuarios, membros: membros, providers: providers, sessoes: sessoes, hasher: hasher}
}

// Executar valida as credenciais e, se corretas, cria uma nova sessão com
// validade de TTLSessao. Retorna ErrCredenciaisInvalidas tanto para email
// inexistente quanto para senha incorreta, e também quando o usuário não tem
// nenhuma agenda vinculada — sem vínculo não há o que operar, e distinguir
// esse caso revelaria que o email existe.
func (uc *LoginProviderUseCase) Executar(input LoginInput) (*LoginOutput, error) {
	u, err := uc.usuarios.BuscarPorEmail(input.Email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		uc.hasher.Verificar(input.Senha, hashDummy)
		return nil, ErrCredenciaisInvalidas
	}

	ok, err := uc.hasher.Verificar(input.Senha, u.SenhaHash)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCredenciaisInvalidas
	}
	if !u.Ativo {
		return nil, ErrUsuarioInativo
	}

	vinculo, err := uc.membros.BuscarPorUsuario(u.ID)
	if err != nil {
		return nil, err
	}
	if vinculo == nil {
		return nil, ErrCredenciaisInvalidas
	}

	p, err := uc.providers.BuscarPorID(vinculo.ProviderID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrCredenciaisInvalidas
	}

	t, err := token.Gerar()
	if err != nil {
		return nil, err
	}

	// A sessão guarda o id do USUÁRIO. Qual agenda ele opera é resolvido a
	// cada requisição pelo vínculo, não congelado no login — é isso que vai
	// permitir trocar de agenda sem refazer a sessão.
	s := session.Nova(token.Hash(t), u.ID, session.TipoProvider, TTLSessao)
	if err := uc.sessoes.Salvar(s); err != nil {
		return nil, err
	}
	uc.sessoes.RemoverExpiradas()

	return &LoginOutput{
		Token:    t,
		ExpiraEm: s.ExpiraEm,
		UserID:   u.ID,
		Nome:     p.Nome,
	}, nil
}
