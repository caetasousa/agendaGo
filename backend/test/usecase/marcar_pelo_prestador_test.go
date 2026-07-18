package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/domain/session"
	ucappointment "agendago/internal/usecase/appointment"
)

func marcacaoPadrao(observacao string) ucappointment.MarcarPeloPrestadorInput {
	return ucappointment.MarcarPeloPrestadorInput{
		ProviderID:    "provider-1",
		Data:          segundaFutura,
		InicioMinutos: 8 * 60,
		Nome:          "Cliente Telefone",
		Observacao:    observacao,
		Agora:         agoraDoTeste,
	}
}

func TestMarcarPeloPrestador(t *testing.T) {
	t.Run("cria convidado só-nome, nasce CONFIRMADO direto, sem nenhum email enviado", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		out, err := amb.marcarPeloPrestador.Executar(marcacaoPadrao(""))
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.Status != appointment.StatusConfirmado {
			t.Errorf("esperava CONFIRMADO, got: %s", out.Status)
		}
		if !out.MarcadoPeloPrestador {
			t.Error("esperava MarcadoPeloPrestador=true na saída")
		}

		a, _ := amb.appointments.BuscarPorID(out.ID)
		if !a.MarcadoPeloPrestador {
			t.Error("esperava MarcadoPeloPrestador=true persistido")
		}
		c, _ := amb.clients.BuscarPorID(a.ClientID)
		if c == nil || c.Nome != "Cliente Telefone" || c.Email != "" || c.Telefone != "" || c.TemConta() {
			t.Errorf("esperava convidado só-nome, got: %+v", c)
		}

		if enviados := amb.mailer.Enviadas(); len(enviados) != 0 {
			t.Errorf("marcação pelo prestador não deveria disparar notificação, got: %d", len(enviados))
		}
	})

	t.Run("observação é persistida e visível na listagem", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		out, err := amb.marcarPeloPrestador.Executar(marcacaoPadrao("cliente prefere corte curto"))
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.Observacao != "cliente prefere corte curto" {
			t.Errorf("esperava observação na saída, got: %q", out.Observacao)
		}

		doPrestador, _ := amb.listar.DoPrestador(ucappointment.ListarInput{UsuarioID: "provider-1", Agora: agoraDoTeste})
		if len(doPrestador.Agendamentos) != 1 || doPrestador.Agendamentos[0].Observacao != "cliente prefere corte curto" {
			t.Errorf("esperava observação visível na listagem do prestador, got: %+v", doPrestador.Agendamentos)
		}
		if !doPrestador.Agendamentos[0].MarcadoPeloPrestador {
			t.Error("esperava MarcadoPeloPrestador=true na listagem")
		}
	})

	t.Run("cada marcação sem cadastro cria um cliente novo, mesmo com o mesmo nome", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		out1, err := amb.marcarPeloPrestador.Executar(marcacaoPadrao(""))
		if err != nil {
			t.Fatalf("primeira marcação falhou: %v", err)
		}
		in2 := marcacaoPadrao("")
		in2.InicioMinutos = 9 * 60
		out2, err := amb.marcarPeloPrestador.Executar(in2)
		if err != nil {
			t.Fatalf("segunda marcação falhou: %v", err)
		}

		a1, _ := amb.appointments.BuscarPorID(out1.ID)
		a2, _ := amb.appointments.BuscarPorID(out2.ID)
		if a1.ClientID == a2.ClientID {
			t.Error("esperava clientes distintos entre marcações sem cadastro")
		}
	})

	t.Run("prestador com a funcionalidade desativada em preferências não consegue marcar", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		amb.prestador.DesativarMarcacaoPeloPrestador()
		amb.providers.Salvar(amb.prestador)

		_, err := amb.marcarPeloPrestador.Executar(marcacaoPadrao(""))
		if err != ucappointment.ErrMarcacaoPeloPrestadorNaoPermitida {
			t.Errorf("esperava ErrMarcacaoPeloPrestadorNaoPermitida, got: %v", err)
		}
	})

	t.Run("nome vazio é rejeitado", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		in := marcacaoPadrao("")
		in.Nome = ""
		if _, err := amb.marcarPeloPrestador.Executar(in); err == nil {
			t.Error("esperava erro com nome vazio")
		}
	})

	t.Run("marca mesmo com a agenda fechada ao público", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		amb.prestador.DesativarAgenda()
		amb.providers.Salvar(amb.prestador)

		// o público não vê slot algum...
		slots, err := amb.consultarSlots.Executar(ucappointment.ConsultarSlotsInput{
			ProviderID: "provider-1", De: segundaFutura, Ate: segundaFutura, Agora: agoraDoTeste,
		})
		if err != nil {
			t.Fatalf("consulta pública falhou: %v", err)
		}
		if len(slots.Dias[0].Slots) != 0 {
			t.Fatalf("agenda fechada não deveria ofertar slots ao público, got: %d", len(slots.Dias[0].Slots))
		}

		// ...mas o dono marca normalmente
		out, err := amb.marcarPeloPrestador.Executar(marcacaoPadrao(""))
		if err != nil {
			t.Fatalf("esperava o dono marcar com agenda fechada, got: %v", err)
		}
		if out.Status != appointment.StatusConfirmado {
			t.Errorf("esperava CONFIRMADO, got: %s", out.Status)
		}
	})

	t.Run("horário ocupado é rejeitado (anti-overbooking)", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		if _, err := amb.marcarPeloPrestador.Executar(marcacaoPadrao("")); err != nil {
			t.Fatalf("primeira marcação falhou: %v", err)
		}
		if _, err := amb.marcarPeloPrestador.Executar(marcacaoPadrao("")); err != ucappointment.ErrHorarioIndisponivel {
			t.Errorf("esperava ErrHorarioIndisponivel na segunda marcação, got: %v", err)
		}
	})

	t.Run("prestador cancela a qualquer momento, mesmo em cima da hora", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		out, err := amb.marcarPeloPrestador.Executar(marcacaoPadrao(""))
		if err != nil {
			t.Fatalf("marcação falhou: %v", err)
		}

		// horário marcado é 08:00 do dia; "agora" bem em cima da hora (mesmo
		// dia, minutos antes do início) violaria a antecedência de 24h de um
		// agendamento comum — aqui não deve importar.
		emCimaDaHora := time.Date(segundaFutura.Year(), segundaFutura.Month(), segundaFutura.Day(), 7, 55, 0, 0, time.UTC)
		if err := amb.transicionar.Cancelar(ucappointment.TransicionarInput{
			AgendamentoID: out.ID, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: emCimaDaHora,
		}); err != nil {
			t.Fatalf("esperava cancelar mesmo em cima da hora, got: %v", err)
		}
		a, _ := amb.appointments.BuscarPorID(out.ID)
		if a.Status != appointment.StatusCancelado {
			t.Errorf("esperava CANCELADO, got: %s", a.Status)
		}
	})

	t.Run("agendamento comum confirmado continua exigindo antecedência mínima", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		out, err := amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60, Agora: agoraDoTeste,
		})
		if err != nil {
			t.Fatalf("solicitação falhou: %v", err)
		}
		if err := amb.transicionar.Confirmar(ucappointment.TransicionarInput{
			AgendamentoID: out.ID, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("confirmar falhou: %v", err)
		}

		emCimaDaHora := time.Date(segundaFutura.Year(), segundaFutura.Month(), segundaFutura.Day(), 7, 55, 0, 0, time.UTC)
		err = amb.transicionar.Cancelar(ucappointment.TransicionarInput{
			AgendamentoID: out.ID, UsuarioID: "provider-1", Tipo: session.TipoProvider, Agora: emCimaDaHora,
		})
		if err != appointment.ErrAntecedenciaInsuficiente {
			t.Errorf("esperava ErrAntecedenciaInsuficiente, got: %v", err)
		}
	})
}
