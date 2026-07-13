package appointment

import "time"

// NotificacaoAgendamento é o payload enviado ao notificador em cada evento do
// ciclo de vida do agendamento.
type NotificacaoAgendamento struct {
	NomePrestador, EmailPrestador string
	NomeCliente, EmailCliente     string
	Data                          time.Time
	InicioMinutos, FimMinutos     int
	// ExpiraEm é o prazo para o prestador confirmar — só preenchido na solicitação.
	ExpiraEm time.Time
	// CanceladoPorPrestador distingue quem cancelou — só preenchido no cancelamento.
	CanceladoPorPrestador bool
	// TokenCancelamento é o token que o convidado usa para cancelar pelo email;
	// só preenchido na confirmação de um agendamento de convidado. Vazio para
	// cliente com conta (que cancela pelo painel).
	TokenCancelamento string
}

// notificadorAgendamento envia os emails dos eventos do ciclo de vida do
// agendamento. Os métodos não retornam erro: o envio é melhor esforço e
// nunca deve falhar a operação que o disparou.
type notificadorAgendamento interface {
	NotificarSolicitacao(n NotificacaoAgendamento)
	NotificarConfirmacao(n NotificacaoAgendamento)
	NotificarRecusa(n NotificacaoAgendamento)
	NotificarCancelamento(n NotificacaoAgendamento)
	NotificarLembrete(n NotificacaoAgendamento)
}
