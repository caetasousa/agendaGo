package auth

import (
	"time"

	"agendago/internal/domain/session"
	"agendago/internal/pkg/token"
)

// ValidarSessaoUseCase valida um token de sessão e devolve a identidade do usuário autenticado.
type ValidarSessaoUseCase struct {
	sessoes repositorioSessao
	membros buscadorMembro
}

// NovoValidarSessaoUseCase cria uma instância de ValidarSessaoUseCase com as dependências injetadas.
func NovoValidarSessaoUseCase(sessoes repositorioSessao, membros buscadorMembro) *ValidarSessaoUseCase {
	return &ValidarSessaoUseCase{sessoes: sessoes, membros: membros}
}

// Executar busca a sessão pelo hash do token informado e retorna a identidade
// do usuário autenticado. Para sessões de prestador resolve também qual agenda
// o usuário opera. Retorna ErrSessaoInvalida se a sessão não existir, já tiver
// expirado, ou — no caso do prestador — se o vínculo com a agenda tiver sumido
// desde o login.
func (uc *ValidarSessaoUseCase) Executar(tokenPuro string) (*Identidade, error) {
	s, err := uc.sessoes.BuscarPorTokenHash(token.Hash(tokenPuro))
	if err != nil {
		return nil, err
	}
	if s == nil || s.Expirada(time.Now()) {
		return nil, ErrSessaoInvalida
	}

	identidade := &Identidade{UserID: s.UserID, Tipo: s.UserType}
	if s.UserType != session.TipoProvider {
		return identidade, nil
	}

	// A agenda é resolvida a cada requisição, e não lida da sessão: assim o
	// acesso revogado vale já na próxima chamada, sem esperar a sessão expirar.
	vinculo, err := uc.membros.BuscarPorUsuario(s.UserID)
	if err != nil {
		return nil, err
	}
	if vinculo == nil {
		return nil, ErrSessaoInvalida
	}
	identidade.ProviderID = vinculo.ProviderID
	identidade.Papel = vinculo.Papel
	return identidade, nil
}
