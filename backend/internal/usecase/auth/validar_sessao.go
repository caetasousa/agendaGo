package auth

import (
	"time"

	"agendago/internal/pkg/token"
)

// ValidarSessaoUseCase valida um token de sessão e devolve a identidade do usuário autenticado.
type ValidarSessaoUseCase struct {
	sessoes repositorioSessao
}

// NovoValidarSessaoUseCase cria uma instância de ValidarSessaoUseCase com o repositório de sessões injetado.
func NovoValidarSessaoUseCase(sessoes repositorioSessao) *ValidarSessaoUseCase {
	return &ValidarSessaoUseCase{sessoes: sessoes}
}

// Executar busca a sessão pelo hash do token informado e retorna a identidade
// do usuário autenticado. Retorna ErrSessaoInvalida se a sessão não existir
// ou já tiver expirado.
func (uc *ValidarSessaoUseCase) Executar(tokenPuro string) (*Identidade, error) {
	s, err := uc.sessoes.BuscarPorTokenHash(token.Hash(tokenPuro))
	if err != nil {
		return nil, err
	}
	if s == nil || s.Expirada(time.Now()) {
		return nil, ErrSessaoInvalida
	}

	return &Identidade{UserID: s.UserID, Tipo: s.UserType}, nil
}
