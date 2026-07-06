package auth

import "agendago/internal/pkg/token"

// LogoutUseCase encerra uma sessão ativa.
type LogoutUseCase struct {
	sessoes repositorioSessao
}

// NovoLogoutUseCase cria uma instância de LogoutUseCase com o repositório de sessões injetado.
func NovoLogoutUseCase(sessoes repositorioSessao) *LogoutUseCase {
	return &LogoutUseCase{sessoes: sessoes}
}

// Executar remove a sessão associada ao token informado. É idempotente: remover
// um token que não corresponde a nenhuma sessão não é erro.
func (uc *LogoutUseCase) Executar(tokenPuro string) error {
	return uc.sessoes.Remover(token.Hash(tokenPuro))
}
