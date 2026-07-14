package usecase_test

import (
	"strings"
	"testing"
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/domain/cancellation"
	"agendago/internal/pkg/token"
	ucappointment "agendago/internal/usecase/appointment"
)

// agendarConvidadoConfirmado cria um agendamento de convidado, confirma-o e
// devolve o id e o token de cancelamento extraído do email de confirmação.
func agendarConvidadoConfirmado(t *testing.T, amb *ambienteAgendamento) (id, token string) {
	t.Helper()

	out, err := amb.solicitarConvidado.Executar(ucappointment.SolicitarConvidadoInput{
		ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60,
		Nome: "Convidado Teste", Email: "convidado@email.com", Telefone: "11999998888",
		Agora: agoraDoTeste,
	})
	if err != nil {
		t.Fatalf("solicitação de convidado falhou: %v", err)
	}

	// descarta os emails da solicitação (que já trazem link de cancelamento)
	// para extrair o token gerado na confirmação
	amb.mailer.Limpar()

	if err := amb.transicionar.Confirmar(ucappointment.TransicionarInput{
		AgendamentoID: out.ID, UsuarioID: "provider-1", Tipo: "provider", Agora: agoraDoTeste,
	}); err != nil {
		t.Fatalf("confirmação falhou: %v", err)
	}

	return out.ID, tokenDoLinkCancelamento(t, amb)
}

// tokenDoLinkCancelamento extrai o token do link /cancelar-agendamento/TOKEN
// presente no email de confirmação capturado pelo mailer.
func tokenDoLinkCancelamento(t *testing.T, amb *ambienteAgendamento) string {
	t.Helper()
	const marcador = "/cancelar-agendamento/"
	for _, msg := range amb.mailer.Enviadas() {
		i := strings.Index(msg.HTML, marcador)
		if i < 0 {
			continue
		}
		resto := msg.HTML[i+len(marcador):]
		fim := strings.IndexAny(resto, "\"' ")
		if fim < 0 {
			fim = len(resto)
		}
		return resto[:fim]
	}
	t.Fatal("nenhum email com link de cancelamento foi enviado")
	return ""
}

func TestCancelamentoConvidado(t *testing.T) {
	t.Run("solicitação de convidado envia email com link de cancelamento e convite de conta", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		if _, err := amb.solicitarConvidado.Executar(ucappointment.SolicitarConvidadoInput{
			ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60,
			Nome: "Convidado Teste", Email: "convidado@email.com", Telefone: "11999998888",
			Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("solicitação de convidado falhou: %v", err)
		}

		var emailConvidado string
		for _, msg := range amb.mailer.Enviadas() {
			if msg.Para == "convidado@email.com" {
				emailConvidado = msg.HTML
			}
		}
		if emailConvidado == "" {
			t.Fatal("esperava email ao convidado na solicitação")
		}
		if !strings.Contains(emailConvidado, "/cancelar-agendamento/") {
			t.Error("esperava link de cancelamento no email da solicitação")
		}
		// o link de cadastro é independente do de cancelamento: leva direto à
		// tela de cadastro, sem passar pela página de cancelamento
		if !strings.Contains(emailConvidado, "/cadastro?pre=") {
			t.Error("esperava link direto de cadastro no email da solicitação")
		}
	})

	t.Run("cancelar por token funciona antes da confirmação do prestador", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		out, err := amb.solicitarConvidado.Executar(ucappointment.SolicitarConvidadoInput{
			ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60,
			Nome: "Convidado Teste", Email: "convidado@email.com", Telefone: "11999998888",
			Agora: agoraDoTeste,
		})
		if err != nil {
			t.Fatalf("solicitação de convidado falhou: %v", err)
		}
		token := tokenDoLinkCancelamento(t, amb)

		if err := amb.cancelarPorToken.Executar(token, agoraDoTeste); err != nil {
			t.Fatalf("esperava cancelar a solicitação pendente, got: %v", err)
		}
		a, _ := amb.appointments.BuscarPorID(out.ID)
		if a.Status != appointment.StatusCancelado {
			t.Errorf("esperava CANCELADO, got: %s", a.Status)
		}
	})

	t.Run("cliente com conta não recebe email de convidado na solicitação", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		if _, err := amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60, Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("solicitação falhou: %v", err)
		}
		for _, msg := range amb.mailer.Enviadas() {
			if msg.Para == "maria@email.com" {
				t.Error("cliente com conta não deveria receber email de convidado na solicitação")
			}
		}
	})

	t.Run("confirmar agendamento de convidado gera token no email", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		_, token := agendarConvidadoConfirmado(t, amb)
		if token == "" {
			t.Fatal("esperava token de cancelamento no email")
		}
	})

	t.Run("confirmar agendamento de cliente com conta NÃO gera token", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		// client-1 tem conta (NovoComConta no ambiente)
		out, err := amb.solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "client-1", ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60, Agora: agoraDoTeste,
		})
		if err != nil {
			t.Fatalf("solicitação falhou: %v", err)
		}
		amb.mailer.Limpar()
		if err := amb.transicionar.Confirmar(ucappointment.TransicionarInput{
			AgendamentoID: out.ID, UsuarioID: "provider-1", Tipo: "provider", Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("confirmação falhou: %v", err)
		}
		for _, msg := range amb.mailer.Enviadas() {
			if strings.Contains(msg.HTML, "/cancelar-agendamento/") {
				t.Error("cliente com conta não deveria receber link de cancelamento")
			}
		}
	})

	t.Run("detalhar por token devolve os dados do agendamento", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		_, token := agendarConvidadoConfirmado(t, amb)

		det, err := amb.cancelarPorToken.Detalhar(token, agoraDoTeste)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if det.NomePrestador != "João Silva" || det.InicioMinutos != 8*60 {
			t.Errorf("detalhe inesperado: %+v", det)
		}
		if !det.PodeCancelar {
			t.Error("esperava podeCancelar=true (bem antes das 24h)")
		}
	})

	t.Run("cancelar por token muda o status para CANCELADO", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		id, token := agendarConvidadoConfirmado(t, amb)

		if err := amb.cancelarPorToken.Executar(token, agoraDoTeste); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		a, _ := amb.appointments.BuscarPorID(id)
		if a.Status != appointment.StatusCancelado {
			t.Errorf("esperava CANCELADO, got: %s", a.Status)
		}
	})

	t.Run("cancelar em cima da hora (<24h) é bloqueado", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		_, token := agendarConvidadoConfirmado(t, amb)

		// 1h antes do início (08:00 de segundaFutura)
		umaHoraAntes := time.Date(2026, 8, 10, 7, 0, 0, 0, time.UTC)
		err := amb.cancelarPorToken.Executar(token, umaHoraAntes)
		if err != appointment.ErrAntecedenciaInsuficiente {
			t.Errorf("esperava ErrAntecedenciaInsuficiente, got: %v", err)
		}
	})

	t.Run("token inválido retorna ErrTokenCancelamentoInvalido", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		if _, err := amb.cancelarPorToken.Detalhar("token-que-nunca-existiu", agoraDoTeste); err != ucappointment.ErrTokenCancelamentoInvalido {
			t.Errorf("esperava ErrTokenCancelamentoInvalido no Detalhar, got: %v", err)
		}
		if err := amb.cancelarPorToken.Executar("token-que-nunca-existiu", agoraDoTeste); err != ucappointment.ErrTokenCancelamentoInvalido {
			t.Errorf("esperava ErrTokenCancelamentoInvalido no Executar, got: %v", err)
		}
	})

	t.Run("token de cancelamento é consumido: usá-lo de novo falha", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		id, token := agendarConvidadoConfirmado(t, amb)

		if err := amb.cancelarPorToken.Executar(token, agoraDoTeste); err != nil {
			t.Fatalf("esperava sucesso no primeiro cancelamento, got: %v", err)
		}
		a, _ := amb.appointments.BuscarPorID(id)
		if a.Status != appointment.StatusCancelado {
			t.Fatalf("esperava CANCELADO, got: %s", a.Status)
		}

		// o mesmo link não cancela de novo — o token já foi consumido
		if err := amb.cancelarPorToken.Executar(token, agoraDoTeste); err != ucappointment.ErrTokenCancelamentoInvalido {
			t.Errorf("esperava ErrTokenCancelamentoInvalido no reuso do token, got: %v", err)
		}
	})

	t.Run("token de cancelamento expirado é rejeitado", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		out, err := amb.solicitarConvidado.Executar(ucappointment.SolicitarConvidadoInput{
			ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60,
			Nome: "Convidado Teste", Email: "convidado@email.com", Telefone: "11999998888",
			Agora: agoraDoTeste,
		})
		if err != nil {
			t.Fatalf("solicitação de convidado falhou: %v", err)
		}

		tokenPuro := "token-cancelamento-vencido"
		// TTL negativo: já nasce expirado, sem precisar esperar ou injetar "agora"
		expirado := cancellation.Novo(token.Hash(tokenPuro), out.ID, -time.Hour)
		if err := amb.cancelamentos.Salvar(expirado); err != nil {
			t.Fatalf("esperava salvar sem erro, got: %v", err)
		}

		if _, err := amb.cancelarPorToken.Detalhar(tokenPuro, agoraDoTeste); err != ucappointment.ErrTokenCancelamentoInvalido {
			t.Errorf("esperava ErrTokenCancelamentoInvalido no Detalhar, got: %v", err)
		}
		if err := amb.cancelarPorToken.Executar(tokenPuro, agoraDoTeste); err != ucappointment.ErrTokenCancelamentoInvalido {
			t.Errorf("esperava ErrTokenCancelamentoInvalido no Executar, got: %v", err)
		}
	})
}
