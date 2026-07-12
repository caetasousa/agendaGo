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

// NotificarConfirmacao avisa o cliente que o prestador confirmou o horário.
func (n *Notificador) NotificarConfirmacao(evento ucappointment.NotificacaoAgendamento) {
	dados := struct{ NomeCliente, NomePrestador, Data, Horario string }{
		NomeCliente:   evento.NomeCliente,
		NomePrestador: evento.NomePrestador,
		Data:          formatarData(evento.Data),
		Horario:       formatarHorario(evento.InicioMinutos),
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
