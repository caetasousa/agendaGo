package auth

import (
	"agendago/internal/domain/passwordreset"
	"agendago/internal/domain/session"
	"agendago/internal/pkg/token"
)

// SolicitarRecuperacaoUseCase inicia a recuperação de senha: gera um token de
// uso único e dispara o email com o link de redefinição.
type SolicitarRecuperacaoUseCase struct {
	providers contaProvider
	clients   contaClient
	resets    repositorioResetSenha
	enviador  enviadorRecuperacao
}

// NovoSolicitarRecuperacaoUseCase cria uma instância de SolicitarRecuperacaoUseCase com as dependências injetadas.
func NovoSolicitarRecuperacaoUseCase(providers contaProvider, clients contaClient, resets repositorioResetSenha, enviador enviadorRecuperacao) *SolicitarRecuperacaoUseCase {
	return &SolicitarRecuperacaoUseCase{providers: providers, clients: clients, resets: resets, enviador: enviador}
}

// Executar busca uma conta (prestador ou cliente com senha) pelo email e, se
// encontrar, emite um token e envia o email de recuperação. Não retorna erro
// nem sinaliza de outra forma se o email não corresponde a nenhuma conta —
// resposta idêntica nos dois casos, para não vazar quais emails existem.
func (uc *SolicitarRecuperacaoUseCase) Executar(email string) error {
	if p, err := uc.providers.BuscarPorEmail(email); err != nil {
		return err
	} else if p != nil && p.Ativo {
		if err := uc.emitir(p.ID, p.Nome, p.Email, session.TipoProvider); err != nil {
			return err
		}
	}

	if c, err := uc.clients.BuscarPorEmail(email); err != nil {
		return err
	} else if c != nil && c.Ativo && c.TemConta() {
		if err := uc.emitir(c.ID, c.Nome, c.Email, session.TipoClient); err != nil {
			return err
		}
	}

	return nil
}

func (uc *SolicitarRecuperacaoUseCase) emitir(userID, nome, email string, tipo session.TipoUsuario) error {
	// invalida qualquer token anterior: só o pedido mais recente fica válido
	if err := uc.resets.RemoverDoUsuario(userID); err != nil {
		return err
	}

	t, err := token.Gerar()
	if err != nil {
		return err
	}

	reset := passwordreset.Novo(token.Hash(t), userID, tipo, TTLRecuperacaoSenha)
	if err := uc.resets.Salvar(reset); err != nil {
		return err
	}

	uc.enviador.EnviarRecuperacaoSenha(email, nome, t, reset.ExpiraEm)
	return nil
}
