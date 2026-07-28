package usecase_test

import (
	"strings"
	"testing"
	"time"

	"agendago/internal/adapter/email"
	"agendago/internal/adapter/security"
	"agendago/internal/domain/admin"
	"agendago/internal/domain/client"
	ucclient "agendago/internal/usecase/client"
	"agendago/test/repository/memoria"
)

// ambienteCadastro monta os usecases de cadastro de cliente sobre repositórios
// em memória, com o Notificador real (síncrono) capturando os emails.
type ambienteCadastro struct {
	solicitar *ucclient.SolicitarCadastroUseCase
	confirmar *ucclient.ConfirmarCadastroUseCase
	clients   *memoria.ClientMemoria
	usuarios  *memoria.UsuarioMemoria
	membros   *memoria.MembroMemoria
	providers *memoria.ProviderMemoria
	admins    *memoria.AdminMemoria
	mailer    *email.MailerMemoria
}

func novoAmbienteCadastro() *ambienteCadastro {
	clients := memoria.NovoClientMemoria()
	usuarios, membros, providers := fakesDePrestador()
	admins := memoria.NovoAdminMemoria()
	pendentes := memoria.NovoSignupMemoria()
	mailer := email.NovaMailerMemoria()
	notificador := email.NovoNotificador(mailer, "http://localhost:5173", time.UTC, email.ExecutorSincrono)
	hasher := security.NovoHasherArgon2id()

	return &ambienteCadastro{
		solicitar: ucclient.NovoSolicitarCadastroUseCase(clients, usuarios, admins, pendentes, notificador, hasher),
		confirmar: ucclient.NovoConfirmarCadastroUseCase(clients, usuarios, admins, pendentes),
		clients:   clients,
		usuarios:  usuarios,
		membros:   membros,
		providers: providers,
		admins:    admins,
		mailer:    mailer,
	}
}

// seedAdmin semeia um admin com o email dado no repositório em memória.
// Compartilhado pelos testes do pacote usecase_test que exercitam a reserva do
// email do admin nos fluxos de cadastro e login social.
func seedAdmin(t *testing.T, admins *memoria.AdminMemoria, email string) {
	t.Helper()
	a, err := admin.Novo("admin-id", email, "hash-qualquer")
	if err != nil {
		t.Fatalf("admin.Novo: %v", err)
	}
	if err := admins.Salvar(a); err != nil {
		t.Fatalf("Salvar admin: %v", err)
	}
}

func TestSolicitarCadastroClienteComEmailDoAdminNaoCriaConta(t *testing.T) {
	amb := novoAmbienteCadastro()
	const emailAdmin = "admin@agendago.com"
	seedAdmin(t, amb.admins, emailAdmin)

	err := amb.solicitar.Executar(ucclient.SolicitarCadastroInput{
		Nome: "Intruso", Email: emailAdmin, Telefone: "11999998888", Senha: "12345678",
	})
	if err != nil {
		t.Fatalf("esperava nil (resposta uniforme), got: %v", err)
	}

	// não pode ter mandado email nem criado pendente: o email do admin é reservado
	if enviados := len(amb.mailer.Enviadas()); enviados != 0 {
		t.Errorf("não deveria enviar email para o email do admin, enviados: %d", enviados)
	}
	if c, _ := amb.clients.BuscarPorEmail(emailAdmin); c != nil {
		t.Error("não deveria existir cliente com o email do admin")
	}
}

// tokenDoEmailCadastro extrai o token do link /confirmar-cadastro?token= do
// último email de confirmação capturado.
func tokenDoEmailCadastro(t *testing.T, mailer *email.MailerMemoria) string {
	t.Helper()
	const marcador = "/confirmar-cadastro?token="
	for _, msg := range mailer.Enviadas() {
		i := strings.Index(msg.HTML, marcador)
		if i < 0 {
			continue
		}
		resto := msg.HTML[i+len(marcador):]
		// o link do prestador ainda traz &tipo=prestador depois do token
		fim := strings.IndexAny(resto, "\"' &")
		if fim < 0 {
			fim = len(resto)
		}
		return resto[:fim]
	}
	t.Fatal("nenhum email de confirmação de cadastro foi enviado")
	return ""
}

func inputCadastro(email string) ucclient.SolicitarCadastroInput {
	return ucclient.SolicitarCadastroInput{
		Nome: "Maria Silva", Email: email, Telefone: "11999998888", Senha: "12345678",
	}
}

func TestSolicitarCadastro(t *testing.T) {
	t.Run("email novo gera pendente e envia confirmação (não cria conta ainda)", func(t *testing.T) {
		amb := novoAmbienteCadastro()
		if err := amb.solicitar.Executar(inputCadastro("maria@email.com")); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		// conta ainda não existe
		if c, _ := amb.clients.BuscarPorEmail("maria@email.com"); c != nil {
			t.Error("conta não deveria existir antes da confirmação")
		}
		if len(amb.mailer.Enviadas()) != 1 {
			t.Fatalf("esperava 1 email, got: %d", len(amb.mailer.Enviadas()))
		}
	})

	t.Run("email de conta existente envia aviso, não pendente", func(t *testing.T) {
		amb := novoAmbienteCadastro()
		conta, _ := client.NovoComConta("c-1", "Maria", "maria@email.com", "hash-existente")
		amb.clients.Salvar(conta)

		if err := amb.solicitar.Executar(inputCadastro("maria@email.com")); err != nil {
			t.Fatalf("esperava sucesso (resposta genérica), got: %v", err)
		}
		enviadas := amb.mailer.Enviadas()
		if len(enviadas) != 1 || !strings.Contains(enviadas[0].Assunto, "já tem uma conta") {
			t.Errorf("esperava email de aviso de conta existente, got: %+v", enviadas)
		}
		// não deve ter criado token de confirmação
		for _, msg := range enviadas {
			if strings.Contains(msg.HTML, "/confirmar-cadastro?token=") {
				t.Error("não deveria emitir token para email de conta existente")
			}
		}
	})

	t.Run("convidado ativo pode virar conta (gera pendente)", func(t *testing.T) {
		amb := novoAmbienteCadastro()
		convidado, _ := client.NovoConvidado("g-1", "Maria", "maria@email.com", "11999998888")
		amb.clients.Salvar(convidado)

		if err := amb.solicitar.Executar(inputCadastro("maria@email.com")); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if tok := tokenDoEmailCadastro(t, amb.mailer); tok == "" {
			t.Error("esperava token de confirmação para convidado")
		}
	})

	t.Run("convidado banido não vira conta pelo cadastro", func(t *testing.T) {
		amb := novoAmbienteCadastro()
		convidado, _ := client.NovoConvidado("g-1", "Maria", "maria@email.com", "11999998888")
		convidado.Banir()
		amb.clients.Salvar(convidado)

		if err := amb.solicitar.Executar(inputCadastro("maria@email.com")); err != nil {
			t.Fatalf("esperava sucesso (silencioso), got: %v", err)
		}
		if len(amb.mailer.Enviadas()) != 0 {
			t.Errorf("esperava zero emails para convidado banido, got: %d", len(amb.mailer.Enviadas()))
		}
	})

	t.Run("email que já é de um prestador envia aviso, não cria pendente", func(t *testing.T) {
		amb := novoAmbienteCadastro()
		criarPrestador(amb.usuarios, amb.membros, amb.providers, "p-1", "João", "joao@email.com", "11999998888", senhaDeTeste)

		if err := amb.solicitar.Executar(inputCadastro("joao@email.com")); err != nil {
			t.Fatalf("esperava sucesso (resposta genérica), got: %v", err)
		}
		enviadas := amb.mailer.Enviadas()
		if len(enviadas) != 1 || !strings.Contains(enviadas[0].Assunto, "já tem uma conta") {
			t.Errorf("esperava aviso de conta existente, got: %+v", enviadas)
		}
		for _, msg := range enviadas {
			if strings.Contains(msg.HTML, "/confirmar-cadastro?token=") {
				t.Error("não deveria emitir token quando o email já é prestador")
			}
		}
	})
}

func TestConfirmarCadastro(t *testing.T) {
	t.Run("email novo: confirma e cria a conta com telefone", func(t *testing.T) {
		amb := novoAmbienteCadastro()
		amb.solicitar.Executar(inputCadastro("maria@email.com"))
		tok := tokenDoEmailCadastro(t, amb.mailer)

		out, err := amb.confirmar.Executar(tok)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		c, _ := amb.clients.BuscarPorEmail("maria@email.com")
		if c == nil || !c.TemConta() {
			t.Fatal("esperava conta criada com senha")
		}
		if c.Telefone != "11999998888" {
			t.Errorf("esperava telefone preenchido, got: %q", c.Telefone)
		}
		if c.SenhaHash == "12345678" {
			t.Error("senha não deveria ser persistida em texto puro")
		}
		if out.ID != c.ID {
			t.Errorf("output ID %s != conta ID %s", out.ID, c.ID)
		}
	})

	t.Run("convidado: confirma preservando o ID (herda agendamentos)", func(t *testing.T) {
		amb := novoAmbienteCadastro()
		convidado, _ := client.NovoConvidado("g-1", "Maria", "maria@email.com", "11999998888")
		amb.clients.Salvar(convidado)

		amb.solicitar.Executar(inputCadastro("maria@email.com"))
		tok := tokenDoEmailCadastro(t, amb.mailer)

		out, err := amb.confirmar.Executar(tok)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.ID != "g-1" {
			t.Errorf("esperava manter o ID do convidado (g-1), got: %s", out.ID)
		}
		c, _ := amb.clients.BuscarPorID("g-1")
		if !c.TemConta() {
			t.Error("convidado deveria ter virado conta")
		}
	})

	t.Run("token inválido retorna ErrCadastroInvalido", func(t *testing.T) {
		amb := novoAmbienteCadastro()
		if _, err := amb.confirmar.Executar("token-que-nunca-existiu"); err != ucclient.ErrCadastroInvalido {
			t.Errorf("esperava ErrCadastroInvalido, got: %v", err)
		}
	})

	t.Run("token é de uso único", func(t *testing.T) {
		amb := novoAmbienteCadastro()
		amb.solicitar.Executar(inputCadastro("maria@email.com"))
		tok := tokenDoEmailCadastro(t, amb.mailer)

		if _, err := amb.confirmar.Executar(tok); err != nil {
			t.Fatalf("primeira confirmação deveria funcionar, got: %v", err)
		}
		if _, err := amb.confirmar.Executar(tok); err != ucclient.ErrCadastroInvalido {
			t.Errorf("esperava ErrCadastroInvalido ao reusar token, got: %v", err)
		}
	})
}
