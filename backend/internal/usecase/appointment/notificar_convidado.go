// Notificação de solicitação ao convidado, compartilhada pelos dois caminhos
// que criam agendamento para quem não tem conta: o link público
// (SolicitarConvidado) e a marcação feita pelo próprio prestador
// (MarcarPeloPrestador).
package appointment

import (
	"agendago/internal/domain/cancellation"
	"agendago/internal/domain/client"
	"agendago/internal/domain/precadastro"
	"agendago/internal/pkg/token"
)

// notificarSolicitacaoAoConvidado envia ao convidado o resumo da solicitação
// com o link de cancelamento por token — sua única via de cancelar sem conta —
// e o link direto para criar a conta, já pré-preenchido. Best-effort: falha ao
// gerar/persistir qualquer um dos tokens só omite o link correspondente, e
// nada aqui falha a reserva já persistida.
func notificarSolicitacaoAoConvidado(
	providerRepo repositorioProvider,
	cancelamentos repositorioCancelamento,
	preCadastros repositorioPreCadastro,
	notificador notificadorAgendamento,
	out *SolicitarOutput,
	convidado *client.Client,
) {
	p, err := providerRepo.BuscarPorID(out.ProviderID)
	if err != nil || p == nil {
		return
	}

	var tokenCancelamento string
	if t, err := token.Gerar(); err == nil {
		if err := cancelamentos.Salvar(cancellation.Novo(token.Hash(t), out.ID, TTLCancelamento)); err == nil {
			tokenCancelamento = t
		}
	}

	var tokenPreCadastro string
	if t, err := token.Gerar(); err == nil {
		pc := precadastro.Novo(token.Hash(t), convidado.Nome, convidado.Email, convidado.Telefone, TTLPreCadastro)
		if err := preCadastros.Salvar(pc); err == nil {
			tokenPreCadastro = t
		}
	}

	notificador.NotificarSolicitacaoConvidado(NotificacaoAgendamento{
		NomePrestador:     p.Nome,
		EmailPrestador:    p.Email,
		NomeCliente:       convidado.Nome,
		EmailCliente:      convidado.Email,
		Data:              out.Data,
		InicioMinutos:     out.InicioMinutos,
		FimMinutos:        out.FimMinutos,
		ExpiraEm:          out.ExpiraEm,
		TokenCancelamento: tokenCancelamento,
		TokenPreCadastro:  tokenPreCadastro,
	})
}
