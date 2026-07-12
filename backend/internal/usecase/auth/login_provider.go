package auth

import (
	"agendago/internal/domain/session"
	"agendago/internal/pkg/token"
)

// LoginProviderUseCase autentica um prestador e cria uma nova sessão.
type LoginProviderUseCase struct {
	providers buscadorProvider
	sessoes   repositorioSessao
	hasher    hasherSenha
}

// NovoLoginProviderUseCase cria uma instância de LoginProviderUseCase com as dependências injetadas.
func NovoLoginProviderUseCase(providers buscadorProvider, sessoes repositorioSessao, hasher hasherSenha) *LoginProviderUseCase {
	return &LoginProviderUseCase{providers: providers, sessoes: sessoes, hasher: hasher}
}

// Executar valida as credenciais do prestador e, se corretas, cria uma nova
// sessão com validade de TTLSessao. Retorna ErrCredenciaisInvalidas tanto para
// email inexistente quanto para senha incorreta.
func (uc *LoginProviderUseCase) Executar(input LoginInput) (*LoginOutput, error) {
	p, err := uc.providers.BuscarPorEmail(input.Email)
	if err != nil {
		return nil, err
	}
	if p == nil {
		uc.hasher.Verificar(input.Senha, hashDummy)
		return nil, ErrCredenciaisInvalidas
	}

	ok, err := uc.hasher.Verificar(input.Senha, p.SenhaHash)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCredenciaisInvalidas
	}
	if !p.Ativo {
		return nil, ErrUsuarioInativo
	}

	t, err := token.Gerar()
	if err != nil {
		return nil, err
	}

	s := session.Nova(token.Hash(t), p.ID, session.TipoProvider, TTLSessao)
	if err := uc.sessoes.Salvar(s); err != nil {
		return nil, err
	}
	uc.sessoes.RemoverExpiradas()

	return &LoginOutput{
		Token:    t,
		ExpiraEm: s.ExpiraEm,
		UserID:   p.ID,
		Nome:     p.Nome,
	}, nil
}
