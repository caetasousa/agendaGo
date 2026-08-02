package lgpd

import (
	"log/slog"
)

// AnonimizarUseCase atende o pedido de exclusão de um cliente.
type AnonimizarUseCase struct {
	clients     repositorioClient
	sessoes     repositorioSessao
	resets      repositorioResetSenha
	identidades repositorioIdentidadeSocial
	auditoria   registradorAuditoria
}

// NovoAnonimizarUseCase cria uma instância de AnonimizarUseCase com os repositórios injetados.
func NovoAnonimizarUseCase(
	clients repositorioClient,
	sessoes repositorioSessao,
	resets repositorioResetSenha,
	identidades repositorioIdentidadeSocial,
	trilha registradorAuditoria,
) *AnonimizarUseCase {
	return &AnonimizarUseCase{clients: clients, sessoes: sessoes, resets: resets, identidades: identidades, auditoria: trilha}
}

// Executar anonimiza o cliente e derruba tudo que dá acesso à conta.
//
// A ordem importa: anonimiza PRIMEIRO, remove acessos depois. Invertendo, uma
// falha no meio deixaria a conta sem sessão mas ainda identificável — estado
// pior, porque a pessoa não consegue nem entrar para pedir de novo.
//
// Os agendamentos ficam. Data, horário e status continuam na agenda do
// prestador, sem dizer de quem eram — ver Anonimizar no domínio.
//
// Retorna ErrClienteNaoEncontrado se a conta não existe e ErrJaAnonimizado se
// já foi removida, para o pedido repetido não parecer ter feito algo.
func (uc *AnonimizarUseCase) Executar(clientID string) error {
	c, err := uc.clients.BuscarPorID(clientID)
	if err != nil {
		return err
	}
	if c == nil {
		return ErrClienteNaoEncontrado
	}
	if c.Anonimizado() {
		return ErrJaAnonimizado
	}

	c.Anonimizar()
	if err := uc.clients.Atualizar(c); err != nil {
		return err
	}

	// Daqui para baixo é melhor-esforço: o dado pessoal já foi apagado, que era
	// o pedido. Uma falha em remover sessão não deve devolver erro e sugerir
	// que a exclusão não aconteceu — mas precisa aparecer no log, porque deixa
	// um acesso vivo para uma conta que deveria estar morta.
	if err := uc.sessoes.RemoverDoUsuario(clientID); err != nil {
		slog.Error("falha ao remover sessões na anonimização", slog.String("erro", err.Error()))
	}
	if err := uc.resets.RemoverDoUsuario(clientID); err != nil {
		slog.Error("falha ao remover tokens de recuperação na anonimização", slog.String("erro", err.Error()))
	}
	if err := uc.identidades.RemoverDoUsuario(clientID); err != nil {
		slog.Error("falha ao remover identidades sociais na anonimização", slog.String("erro", err.Error()))
	}

	uc.registrar(clientID)
	return nil
}
