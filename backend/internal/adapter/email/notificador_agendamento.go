package email

import (
	ucappointment "agendago/internal/usecase/appointment"
)

// NotificarSolicitacao avisa o prestador que recebeu um novo pedido de horário.
func (n *Notificador) NotificarSolicitacao(evento ucappointment.NotificacaoAgendamento) {
	dados := struct {
		NomePrestador, NomeCliente, Data, Horario, ExpiraEm, Link string
	}{
		NomePrestador: evento.NomePrestador,
		NomeCliente:   evento.NomeCliente,
		Data:          formatarData(evento.Data),
		Horario:       formatarHorario(evento.InicioMinutos),
		ExpiraEm:      evento.ExpiraEm.In(n.fuso).Format("02/01/2006 15:04"),
		Link:          n.urlFrontend + "/painel/agendamentos",
	}
	n.enviar(evento.EmailPrestador, evento.NomePrestador, "Novo pedido de horário — agendaGo", "solicitacao_prestador.html", dados)
}

// NotificarSolicitacaoConvidado envia ao convidado o resumo do pedido que ele
// acabou de fazer: o horário aguarda a confirmação do prestador, com o link
// para cancelar por token (sua única via, já que não tem conta) e o link
// direto para criar uma conta — já pré-preenchida, ação independente do
// cancelamento.
func (n *Notificador) NotificarSolicitacaoConvidado(evento ucappointment.NotificacaoAgendamento) {
	linkCancelamento := ""
	if evento.TokenCancelamento != "" {
		linkCancelamento = n.urlFrontend + "/cancelar-agendamento/" + evento.TokenCancelamento
	}
	linkCadastro := ""
	if evento.TokenPreCadastro != "" {
		linkCadastro = n.urlFrontend + "/cadastro?pre=" + evento.TokenPreCadastro
	}
	dados := struct {
		NomeCliente, NomePrestador, Data, Horario, ExpiraEm, LinkCancelamento, LinkCadastro string
	}{
		NomeCliente:      evento.NomeCliente,
		NomePrestador:    evento.NomePrestador,
		Data:             formatarData(evento.Data),
		Horario:          formatarHorario(evento.InicioMinutos),
		ExpiraEm:         evento.ExpiraEm.In(n.fuso).Format("02/01/2006 15:04"),
		LinkCancelamento: linkCancelamento,
		LinkCadastro:     linkCadastro,
	}
	n.enviar(evento.EmailCliente, evento.NomeCliente, "Recebemos sua solicitação — agendaGo", "solicitacao_convidado.html", dados)
}

// NotificarConfirmacao avisa o cliente que o prestador confirmou o horário.
// Quando há token de cancelamento (agendamento de convidado), inclui o link
// para o convidado poder cancelar sem conta, e o link direto para criar
// conta já pré-preenchida.
func (n *Notificador) NotificarConfirmacao(evento ucappointment.NotificacaoAgendamento) {
	linkCancelamento := ""
	if evento.TokenCancelamento != "" {
		linkCancelamento = n.urlFrontend + "/cancelar-agendamento/" + evento.TokenCancelamento
	}
	linkCadastro := ""
	if evento.TokenPreCadastro != "" {
		linkCadastro = n.urlFrontend + "/cadastro?pre=" + evento.TokenPreCadastro
	}
	dados := struct{ NomeCliente, NomePrestador, Data, Horario, LinkCancelamento, LinkCadastro string }{
		NomeCliente:      evento.NomeCliente,
		NomePrestador:    evento.NomePrestador,
		Data:             formatarData(evento.Data),
		Horario:          formatarHorario(evento.InicioMinutos),
		LinkCancelamento: linkCancelamento,
		LinkCadastro:     linkCadastro,
	}
	n.enviar(evento.EmailCliente, evento.NomeCliente, "Agendamento confirmado — agendaGo", "confirmado_cliente.html", dados)
}

// NotificarRecusa avisa o cliente que o prestador não pôde confirmar o horário.
func (n *Notificador) NotificarRecusa(evento ucappointment.NotificacaoAgendamento) {
	dados := struct{ NomeCliente, NomePrestador, Data, Horario string }{
		NomeCliente:   evento.NomeCliente,
		NomePrestador: evento.NomePrestador,
		Data:          formatarData(evento.Data),
		Horario:       formatarHorario(evento.InicioMinutos),
	}
	n.enviar(evento.EmailCliente, evento.NomeCliente, "Horário não confirmado — agendaGo", "recusado_cliente.html", dados)
}

// NotificarCancelamento avisa a outra parte que o agendamento foi cancelado.
func (n *Notificador) NotificarCancelamento(evento ucappointment.NotificacaoAgendamento) {
	canceladoPor := "O cliente"
	destino, nomeDestino := evento.EmailPrestador, evento.NomePrestador
	if evento.CanceladoPorPrestador {
		canceladoPor = "O prestador"
		destino, nomeDestino = evento.EmailCliente, evento.NomeCliente
	}
	dados := struct{ NomeDestinatario, CanceladoPor, Data, Horario string }{
		NomeDestinatario: nomeDestino,
		CanceladoPor:     canceladoPor,
		Data:             formatarData(evento.Data),
		Horario:          formatarHorario(evento.InicioMinutos),
	}
	n.enviar(destino, nomeDestino, "Agendamento cancelado — agendaGo", "cancelado.html", dados)
}

// NotificarLembrete avisa o cliente que o atendimento confirmado é no dia seguinte.
func (n *Notificador) NotificarLembrete(evento ucappointment.NotificacaoAgendamento) {
	dados := struct{ NomeCliente, NomePrestador, Data, Horario string }{
		NomeCliente:   evento.NomeCliente,
		NomePrestador: evento.NomePrestador,
		Data:          formatarData(evento.Data),
		Horario:       formatarHorario(evento.InicioMinutos),
	}
	n.enviar(evento.EmailCliente, evento.NomeCliente, "Lembrete: atendimento amanhã — agendaGo", "lembrete_cliente.html", dados)
}
