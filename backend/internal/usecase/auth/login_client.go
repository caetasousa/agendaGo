package auth

import (
	"agendago/internal/domain/session"
	"agendago/internal/pkg/token"
)

// LoginClientUseCase autentica um cliente com conta e cria uma nova sessão.
type LoginClientUseCase struct {
	clients buscadorClient
	sessoes repositorioSessao
	hasher  hasherSenha
}

// NovoLoginClientUseCase cria uma instância de LoginClientUseCase com as dependências injetadas.
func NovoLoginClientUseCase(clients buscadorClient, sessoes repositorioSessao, hasher hasherSenha) *LoginClientUseCase {
	return &LoginClientUseCase{clients: clients, sessoes: sessoes, hasher: hasher}
}

// Executar valida as credenciais do cliente e, se corretas, cria uma nova
// sessão com validade de TTLSessao. Retorna ErrCredenciaisInvalidas para email
// inexistente, senha incorreta, ou cliente convidado (sem conta).
func (uc *LoginClientUseCase) Executar(input LoginInput) (*LoginOutput, error) {
	c, err := uc.clients.BuscarPorEmail(input.Email)
	if err != nil {
		return nil, err
	}
	if c == nil || !c.TemConta() {
		uc.hasher.Verificar(input.Senha, hashDummy)
		return nil, ErrCredenciaisInvalidas
	}

	ok, err := uc.hasher.Verificar(input.Senha, c.SenhaHash)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCredenciaisInvalidas
	}
	if !c.Ativo {
		return nil, ErrUsuarioInativo
	}

	t, err := token.Gerar()
	if err != nil {
		return nil, err
	}

	s := session.Nova(token.Hash(t), c.ID, session.TipoClient, TTLSessao)
	if err := uc.sessoes.Salvar(s); err != nil {
		return nil, err
	}
	uc.sessoes.RemoverExpiradas()

	return &LoginOutput{
		Token:    t,
		ExpiraEm: s.ExpiraEm,
		UserID:   c.ID,
		Nome:     c.Nome,
	}, nil
}
