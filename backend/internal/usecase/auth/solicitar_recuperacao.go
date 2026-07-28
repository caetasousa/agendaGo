package auth

import (
	"agendago/internal/domain/passwordreset"
	"agendago/internal/domain/session"
	"agendago/internal/pkg/token"
)

// SolicitarRecuperacaoUseCase inicia a recuperação de senha: gera um token de
// uso único e dispara o email com o link de redefinição.
type SolicitarRecuperacaoUseCase struct {
	usuarios  contaUsuario
	membros   buscadorMembro
	providers buscadorProvider
	clients   contaClient
	resets    repositorioResetSenha
	enviador  enviadorRecuperacao
}

// NovoSolicitarRecuperacaoUseCase cria uma instância de SolicitarRecuperacaoUseCase com as dependências injetadas.
func NovoSolicitarRecuperacaoUseCase(
	usuarios contaUsuario,
	membros buscadorMembro,
	providers buscadorProvider,
	clients contaClient,
	resets repositorioResetSenha,
	enviador enviadorRecuperacao,
) *SolicitarRecuperacaoUseCase {
	return &SolicitarRecuperacaoUseCase{usuarios: usuarios, membros: membros, providers: providers, clients: clients, resets: resets, enviador: enviador}
}

// Executar busca uma conta (prestador ou cliente com senha) pelo email e, se
// encontrar, emite um token e envia o email de recuperação. Não retorna erro
// nem sinaliza de outra forma se o email não corresponde a nenhuma conta —
// resposta idêntica nos dois casos, para não vazar quais emails existem.
func (uc *SolicitarRecuperacaoUseCase) Executar(email string) error {
	if u, err := uc.usuarios.BuscarPorEmail(email); err != nil {
		return err
	} else if u != nil && u.Ativo {
		nome, err := uc.nomeDaAgenda(u.ID)
		if err != nil {
			return err
		}
		if err := uc.emitir(u.ID, nome, u.Email, session.TipoProvider); err != nil {
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

// nomeDaAgenda devolve o nome exibido no email de recuperação. Ele vem da
// agenda, não da conta: quem loga não tem nome próprio no modelo. Sem vínculo
// ou sem agenda o email sai sem saudação personalizada, em vez de falhar — a
// recuperação de senha é o pior momento para barrar alguém por um dado
// cosmético.
func (uc *SolicitarRecuperacaoUseCase) nomeDaAgenda(usuarioID string) (string, error) {
	vinculo, err := uc.membros.BuscarPorUsuario(usuarioID)
	if err != nil {
		return "", err
	}
	if vinculo == nil {
		return "", nil
	}
	p, err := uc.providers.BuscarPorID(vinculo.ProviderID)
	if err != nil {
		return "", err
	}
	if p == nil {
		return "", nil
	}
	return p.Nome, nil
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
