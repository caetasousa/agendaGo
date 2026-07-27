package usecase_test

import (
	"strings"
	"testing"
	"time"

	"agendago/internal/adapter/email"
	"agendago/internal/adapter/security"
	"agendago/internal/domain/client"
	ucclient "agendago/internal/usecase/client"
	ucprovider "agendago/internal/usecase/provider"
	"agendago/test/repository/memoria"
)

// ambienteCadastroProvider monta os usecases de cadastro de prestador sobre
// repositórios em memória, com o Notificador real (síncrono) capturando os emails.
type ambienteCadastroProvider struct {
	solicitar *ucprovider.SolicitarCadastroUseCase
	confirmar *ucprovider.ConfirmarCadastroUseCase
	providers *memoria.ProviderMemoria
	clients   *memoria.ClientMemoria
	admins    *memoria.AdminMemoria
	mailer    *email.MailerMemoria
}

func novoAmbienteCadastroProvider() *ambienteCadastroProvider {
	providers := memoria.NovoProviderMemoria()
	clients := memoria.NovoClientMemoria()
	admins := memoria.NovoAdminMemoria()
	pendentes := memoria.NovoSignupMemoria()
	mailer := email.NovaMailerMemoria()
	notificador := email.NovoNotificador(mailer, "http://localhost:5173", time.UTC, email.ExecutorSincrono)
	hasher := security.NovoHasherArgon2id()

	return &ambienteCadastroProvider{
		solicitar: ucprovider.NovoSolicitarCadastroUseCase(providers, clients, admins, pendentes, notificador, hasher),
		confirmar: ucprovider.NovoConfirmarCadastroUseCase(providers, clients, admins, pendentes),
		providers: providers,
		clients:   clients,
		admins:    admins,
		mailer:    mailer,
	}
}

func TestSolicitarCadastroProviderComEmailDoAdminNaoCriaConta(t *testing.T) {
	amb := novoAmbienteCadastroProvider()
	const emailAdmin = "admin@agendago.com"
	seedAdmin(t, amb.admins, emailAdmin)

	err := amb.solicitar.Executar(ucprovider.SolicitarCadastroInput{
		Nome: "Intruso", Email: emailAdmin, Telefone: "11999998888", Senha: "12345678",
	})
	if err != nil {
		t.Fatalf("esperava nil (resposta uniforme), got: %v", err)
	}

	if enviados := len(amb.mailer.Enviadas()); enviados != 0 {
		t.Errorf("não deveria enviar email para o email do admin, enviados: %d", enviados)
	}
	if p, _ := amb.providers.BuscarPorEmail(emailAdmin); p != nil {
		t.Error("não deveria existir prestador com o email do admin")
	}
}

// tokenDoEmailCadastroProvider extrai o token do link /confirmar-cadastro?token=
// do último email de confirmação capturado.
func tokenDoEmailCadastroProvider(t *testing.T, mailer *email.MailerMemoria) string {
	t.Helper()
	const marcador = "/confirmar-cadastro?token="
	for _, msg := range mailer.Enviadas() {
		i := strings.Index(msg.HTML, marcador)
		if i < 0 {
			continue
		}
		resto := msg.HTML[i+len(marcador):]
		fim := strings.IndexAny(resto, "\"' &")
		if fim < 0 {
			fim = len(resto)
		}
		return resto[:fim]
	}
	t.Fatal("nenhum email de confirmação de cadastro foi enviado")
	return ""
}

func inputCadastroProvider(email string) ucprovider.SolicitarCadastroInput {
	return ucprovider.SolicitarCadastroInput{
		Nome: "João Silva", Email: email, Telefone: "11999998888", Senha: "12345678",
	}
}

func TestSolicitarCadastroProvider(t *testing.T) {
	t.Run("email novo gera pendente e envia confirmação (não cria conta ainda)", func(t *testing.T) {
		amb := novoAmbienteCadastroProvider()
		if err := amb.solicitar.Executar(inputCadastroProvider("joao@email.com")); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if p, _ := amb.providers.BuscarPorEmail("joao@email.com"); p != nil {
			t.Error("conta não deveria existir antes da confirmação")
		}
		if len(amb.mailer.Enviadas()) != 1 {
			t.Fatalf("esperava 1 email, got: %d", len(amb.mailer.Enviadas()))
		}
	})

	t.Run("email já em uso responde igual (resposta sempre a mesma)", func(t *testing.T) {
		amb := novoAmbienteCadastroProvider()
		input := inputCadastroProvider("joao@email.com")

		if err := amb.solicitar.Executar(input); err != nil {
			t.Fatalf("esperava sucesso na primeira solicitação, got: %v", err)
		}
		tok := tokenDoEmailCadastroProvider(t, amb.mailer)
		if _, err := amb.confirmar.Executar(tok); err != nil {
			t.Fatalf("esperava confirmação bem-sucedida, got: %v", err)
		}

		if err := amb.solicitar.Executar(input); err != nil {
			t.Fatalf("esperava sucesso (resposta genérica) na segunda, got: %v", err)
		}
		enviadas := amb.mailer.Enviadas()
		if len(enviadas) != 2 || !strings.Contains(enviadas[1].Assunto, "já tem uma conta") {
			t.Errorf("esperava aviso de conta existente na segunda tentativa, got: %+v", enviadas)
		}
	})

	t.Run("email que já pertence a um cliente/convidado envia aviso, não cria pendente", func(t *testing.T) {
		amb := novoAmbienteCadastroProvider()
		convidado, _ := client.NovoConvidado("c-1", "Maria", "maria@email.com", "11999998888")
		amb.clients.Salvar(convidado)

		if err := amb.solicitar.Executar(inputCadastroProvider("maria@email.com")); err != nil {
			t.Fatalf("esperava sucesso (resposta genérica), got: %v", err)
		}
		enviadas := amb.mailer.Enviadas()
		if len(enviadas) != 1 || !strings.Contains(enviadas[0].Assunto, "já tem uma conta") {
			t.Errorf("esperava aviso de conta existente, got: %+v", enviadas)
		}
		for _, msg := range enviadas {
			if strings.Contains(msg.HTML, "/confirmar-cadastro?token=") {
				t.Error("não deveria emitir token quando o email já pertence a um cliente")
			}
		}
	})

	t.Run("persiste a senha com hash, nunca em texto puro", func(t *testing.T) {
		amb := novoAmbienteCadastroProvider()
		amb.solicitar.Executar(inputCadastroProvider("joao@email.com"))
		tok := tokenDoEmailCadastroProvider(t, amb.mailer)
		amb.confirmar.Executar(tok)

		p, _ := amb.providers.BuscarPorEmail("joao@email.com")
		if p.SenhaHash == "12345678" {
			t.Error("senha não deveria ser persistida em texto puro")
		}
		if p.SenhaHash == "" {
			t.Error("hash de senha não deveria ser vazio")
		}
	})
}

func TestConfirmarCadastroProvider(t *testing.T) {
	t.Run("token válido confirma e cria a conta", func(t *testing.T) {
		amb := novoAmbienteCadastroProvider()
		amb.solicitar.Executar(inputCadastroProvider("joao@email.com"))
		tok := tokenDoEmailCadastroProvider(t, amb.mailer)

		out, err := amb.confirmar.Executar(tok)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		p, _ := amb.providers.BuscarPorEmail("joao@email.com")
		if p == nil {
			t.Fatal("esperava prestador criado")
		}
		if out.ID != p.ID || out.Nome != p.Nome || out.Email != p.Email {
			t.Errorf("output %+v não confere com o prestador criado %+v", out, p)
		}
	})

	t.Run("token inválido retorna ErrCadastroInvalido", func(t *testing.T) {
		amb := novoAmbienteCadastroProvider()
		if _, err := amb.confirmar.Executar("token-que-nunca-existiu"); err != ucprovider.ErrCadastroInvalido {
			t.Errorf("esperava ErrCadastroInvalido, got: %v", err)
		}
	})

	t.Run("token é de uso único", func(t *testing.T) {
		amb := novoAmbienteCadastroProvider()
		amb.solicitar.Executar(inputCadastroProvider("joao@email.com"))
		tok := tokenDoEmailCadastroProvider(t, amb.mailer)

		if _, err := amb.confirmar.Executar(tok); err != nil {
			t.Fatalf("primeira confirmação deveria funcionar, got: %v", err)
		}
		if _, err := amb.confirmar.Executar(tok); err != ucprovider.ErrCadastroInvalido {
			t.Errorf("esperava ErrCadastroInvalido ao reusar token, got: %v", err)
		}
	})

	t.Run("token de cadastro de cliente não confirma conta de prestador", func(t *testing.T) {
		// mesmo repositório de pendentes para os dois lados: o que se testa é
		// que o Tipo gravado no banco (não o endpoint chamado) decide a conta
		pendentes := memoria.NovoSignupMemoria()
		providers := memoria.NovoProviderMemoria()
		clients := memoria.NovoClientMemoria()
		admins := memoria.NovoAdminMemoria()
		mailer := email.NovaMailerMemoria()
		notificador := email.NovoNotificador(mailer, "http://localhost:5173", time.UTC, email.ExecutorSincrono)
		hasher := security.NovoHasherArgon2id()

		solicitarCliente := ucclient.NovoSolicitarCadastroUseCase(clients, providers, admins, pendentes, notificador, hasher)
		confirmarProvider := ucprovider.NovoConfirmarCadastroUseCase(providers, clients, admins, pendentes)

		solicitarCliente.Executar(ucclient.SolicitarCadastroInput{
			Nome: "Maria", Email: "maria@email.com", Telefone: "11999998888", Senha: "12345678",
		})
		tok := tokenDoEmailCadastroProvider(t, mailer)

		if _, err := confirmarProvider.Executar(tok); err != ucprovider.ErrCadastroInvalido {
			t.Errorf("esperava ErrCadastroInvalido: token de cliente não deve confirmar cadastro de prestador, got: %v", err)
		}
	})
}
