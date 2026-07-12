package auth

import (
	"agendago/internal/domain/session"
	"agendago/internal/pkg/token"
)

// LoginAdminUseCase autentica um administrador e cria uma nova sessão.
type LoginAdminUseCase struct {
	admins  buscadorAdmin
	sessoes repositorioSessao
	hasher  hasherSenha
}

// NovoLoginAdminUseCase cria uma instância de LoginAdminUseCase com as dependências injetadas.
func NovoLoginAdminUseCase(admins buscadorAdmin, sessoes repositorioSessao, hasher hasherSenha) *LoginAdminUseCase {
	return &LoginAdminUseCase{admins: admins, sessoes: sessoes, hasher: hasher}
}

// Executar valida as credenciais do admin e, se corretas, cria uma sessão com
// validade de TTLSessao. Retorna ErrCredenciaisInvalidas tanto para email
// inexistente quanto para senha incorreta.
func (uc *LoginAdminUseCase) Executar(input LoginInput) (*LoginOutput, error) {
	a, err := uc.admins.BuscarPorEmail(input.Email)
	if err != nil {
		return nil, err
	}
	if a == nil {
		uc.hasher.Verificar(input.Senha, hashDummy)
		return nil, ErrCredenciaisInvalidas
	}

	ok, err := uc.hasher.Verificar(input.Senha, a.SenhaHash)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCredenciaisInvalidas
	}

	t, err := token.Gerar()
	if err != nil {
		return nil, err
	}

	s := session.Nova(token.Hash(t), a.ID, session.TipoAdmin, TTLSessao)
	if err := uc.sessoes.Salvar(s); err != nil {
		return nil, err
	}
	uc.sessoes.RemoverExpiradas()

	return &LoginOutput{
		Token:    t,
		ExpiraEm: s.ExpiraEm,
		UserID:   a.ID,
		Nome:     "Admin",
	}, nil
}
