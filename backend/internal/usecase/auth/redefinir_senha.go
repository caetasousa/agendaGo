package auth

import (
	"time"

	"agendago/internal/domain/session"
	"agendago/internal/pkg/token"
)

// RedefinirSenhaUseCase conclui a recuperação de senha: consome o token e
// grava a nova senha.
type RedefinirSenhaUseCase struct {
	providers contaProvider
	clients   contaClient
	resets    repositorioResetSenha
	sessoes   repositorioSessao
	hasher    hasherSenha
}

// NovoRedefinirSenhaUseCase cria uma instância de RedefinirSenhaUseCase com as dependências injetadas.
func NovoRedefinirSenhaUseCase(providers contaProvider, clients contaClient, resets repositorioResetSenha, sessoes repositorioSessao, hasher hasherSenha) *RedefinirSenhaUseCase {
	return &RedefinirSenhaUseCase{providers: providers, clients: clients, resets: resets, sessoes: sessoes, hasher: hasher}
}

// Executar consome o token de recuperação (uso único) e, se válido e não
// expirado, grava a nova senha e revoga todas as sessões ativas do usuário —
// uma redefinição de senha é motivo para exigir login de novo em qualquer
// dispositivo. Retorna ErrTokenRecuperacaoInvalido tanto para token
// inexistente quanto para token expirado.
func (uc *RedefinirSenhaUseCase) Executar(tokenPuro, novaSenha string) error {
	reset, err := uc.resets.Consumir(token.Hash(tokenPuro))
	if err != nil {
		return err
	}
	if reset == nil || reset.Expirado(time.Now()) {
		return ErrTokenRecuperacaoInvalido
	}

	hash, err := uc.hasher.Gerar(novaSenha)
	if err != nil {
		return err
	}

	switch reset.UserType {
	case session.TipoProvider:
		if err := uc.providers.AtualizarSenha(reset.UserID, hash); err != nil {
			return err
		}
	case session.TipoClient:
		if err := uc.clients.AtualizarSenha(reset.UserID, hash); err != nil {
			return err
		}
	default:
		return ErrTokenRecuperacaoInvalido
	}

	if err := uc.sessoes.RemoverDoUsuario(reset.UserID); err != nil {
		return err
	}
	uc.resets.RemoverExpirados()
	return nil
}
