package usecase_test

import (
	"strings"
	"testing"
	"time"

	"agendago/internal/domain/client"
	"agendago/internal/domain/precadastro"
	"agendago/internal/pkg/token"
	ucappointment "agendago/internal/usecase/appointment"
	ucclient "agendago/internal/usecase/client"
)

// tokenDoLinkCadastro extrai o token do link /cadastro?pre=TOKEN presente no
// primeiro email capturado que o contém.
func tokenDoLinkCadastro(t *testing.T, amb *ambienteAgendamento) string {
	t.Helper()
	const marcador = "/cadastro?pre="
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
	t.Fatal("nenhum email com link de cadastro foi enviado")
	return ""
}

func TestPreCadastroConvidado(t *testing.T) {
	t.Run("solicitação de convidado já traz o link de cadastro pré-preenchido", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		if _, err := amb.solicitarConvidado.Executar(ucappointment.SolicitarConvidadoInput{
			ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60,
			Nome: "Convidado Teste", Email: "convidado@email.com", Telefone: "11999998888",
			Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("solicitação de convidado falhou: %v", err)
		}

		tokenPreCadastro := tokenDoLinkCadastro(t, amb)
		out, err := amb.consultarPreCadastro.Executar(tokenPreCadastro)
		if err != nil {
			t.Fatalf("esperava consultar com sucesso, got: %v", err)
		}
		if out.Nome != "Convidado Teste" || out.Email != "convidado@email.com" || out.Telefone != "11999998888" {
			t.Errorf("dados inesperados: %+v", out)
		}
	})

	t.Run("confirmação também traz um novo link de cadastro pré-preenchido", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)
		_, _ = agendarConvidadoConfirmado(t, amb)

		tokenPreCadastro := tokenDoLinkCadastro(t, amb)
		out, err := amb.consultarPreCadastro.Executar(tokenPreCadastro)
		if err != nil {
			t.Fatalf("esperava consultar com sucesso, got: %v", err)
		}
		if out.Nome != "Convidado Teste" || out.Email != "convidado@email.com" {
			t.Errorf("dados inesperados: %+v", out)
		}
	})

	t.Run("consultar o mesmo token de pré-cadastro mais de uma vez não invalida o link", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		if _, err := amb.solicitarConvidado.Executar(ucappointment.SolicitarConvidadoInput{
			ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60,
			Nome: "Convidado Teste", Email: "convidado@email.com", Telefone: "11999998888",
			Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("solicitação de convidado falhou: %v", err)
		}

		tokenPreCadastro := tokenDoLinkCadastro(t, amb)
		if _, err := amb.consultarPreCadastro.Executar(tokenPreCadastro); err != nil {
			t.Fatalf("primeira consulta deveria funcionar, got: %v", err)
		}
		if _, err := amb.consultarPreCadastro.Executar(tokenPreCadastro); err != nil {
			t.Errorf("segunda consulta também deveria funcionar (só o submit final consome), got: %v", err)
		}
	})

	t.Run("concluir o pré-cadastro cria a conta direto, sem segunda confirmação por email", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		if _, err := amb.solicitarConvidado.Executar(ucappointment.SolicitarConvidadoInput{
			ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60,
			Nome: "Convidado Teste", Email: "convidado@email.com", Telefone: "11999998888",
			Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("solicitação de convidado falhou: %v", err)
		}

		tokenPreCadastro := tokenDoLinkCadastro(t, amb)
		out, err := amb.concluirPreCadastro.Executar(ucclient.ConcluirPreCadastroInput{
			Token: tokenPreCadastro, Senha: "SenhaForte123",
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.Email != "convidado@email.com" || out.Nome != "Convidado Teste" {
			t.Errorf("dados inesperados: %+v", out)
		}

		conta, _ := amb.clients.BuscarPorEmail("convidado@email.com")
		if conta == nil || !conta.TemConta() {
			t.Fatal("esperava conta criada com senha definida")
		}
	})

	t.Run("token de pré-cadastro é de uso único na conclusão", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		if _, err := amb.solicitarConvidado.Executar(ucappointment.SolicitarConvidadoInput{
			ProviderID: "provider-1", Data: segundaFutura, InicioMinutos: 8 * 60,
			Nome: "Convidado Teste", Email: "convidado@email.com", Telefone: "11999998888",
			Agora: agoraDoTeste,
		}); err != nil {
			t.Fatalf("solicitação de convidado falhou: %v", err)
		}

		tokenPreCadastro := tokenDoLinkCadastro(t, amb)
		if _, err := amb.concluirPreCadastro.Executar(ucclient.ConcluirPreCadastroInput{
			Token: tokenPreCadastro, Senha: "SenhaForte123",
		}); err != nil {
			t.Fatalf("primeira conclusão deveria funcionar, got: %v", err)
		}
		if _, err := amb.concluirPreCadastro.Executar(ucclient.ConcluirPreCadastroInput{
			Token: tokenPreCadastro, Senha: "OutraSenha123",
		}); err != ucclient.ErrPreCadastroInvalido {
			t.Errorf("segunda conclusão deveria falhar com ErrPreCadastroInvalido, got: %v", err)
		}
	})

	t.Run("cliente com conta não recebe link de cadastro na confirmação", func(t *testing.T) {
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
			if strings.Contains(msg.HTML, "/cadastro?pre=") {
				t.Error("cliente com conta não deveria receber link de pré-cadastro")
			}
		}
	})

	t.Run("token de pré-cadastro inválido na consulta", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		if _, err := amb.consultarPreCadastro.Executar("token-que-nunca-existiu"); err != ucclient.ErrPreCadastroInvalido {
			t.Errorf("esperava ErrPreCadastroInvalido, got: %v", err)
		}
	})

	t.Run("token de pré-cadastro expirado é rejeitado na consulta e na conclusão", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		tokenPuro := "token-vencido-precadastro"
		// TTL negativo: já nasce expirado, sem precisar esperar ou injetar "agora"
		expirado := precadastro.Novo(token.Hash(tokenPuro), "Convidado Vencido", "vencido@email.com", "11999998888", -time.Hour)
		if err := amb.preCadastros.Salvar(expirado); err != nil {
			t.Fatalf("esperava salvar sem erro, got: %v", err)
		}

		if _, err := amb.consultarPreCadastro.Executar(tokenPuro); err != ucclient.ErrPreCadastroInvalido {
			t.Errorf("esperava ErrPreCadastroInvalido na consulta, got: %v", err)
		}

		if _, err := amb.concluirPreCadastro.Executar(ucclient.ConcluirPreCadastroInput{
			Token: tokenPuro, Senha: "SenhaForte123",
		}); err != ucclient.ErrPreCadastroInvalido {
			t.Errorf("esperava ErrPreCadastroInvalido na conclusão, got: %v", err)
		}
	})

	t.Run("materializarConta rejeita convidado banido, mesmo com token de pré-cadastro válido", func(t *testing.T) {
		amb := novoAmbienteAgendamento(t)

		banido, err := client.NovoConvidado("convidado-banido", "Convidado Banido", "banido@email.com", "11999998888")
		if err != nil {
			t.Fatalf("esperava criar convidado, got: %v", err)
		}
		banido.Ativo = false
		if err := amb.clients.Salvar(banido); err != nil {
			t.Fatalf("esperava salvar convidado banido, got: %v", err)
		}

		tokenPuro := "token-convidado-banido"
		pc := precadastro.Novo(token.Hash(tokenPuro), banido.Nome, banido.Email, banido.Telefone, time.Hour)
		if err := amb.preCadastros.Salvar(pc); err != nil {
			t.Fatalf("esperava salvar pré-cadastro, got: %v", err)
		}

		if _, err := amb.concluirPreCadastro.Executar(ucclient.ConcluirPreCadastroInput{
			Token: tokenPuro, Senha: "SenhaForte123",
		}); err != ucclient.ErrCadastroInvalido {
			t.Errorf("esperava ErrCadastroInvalido (banimento não é revertido por cadastro), got: %v", err)
		}
	})
}
