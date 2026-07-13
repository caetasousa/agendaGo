package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/email"
	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	"agendago/internal/domain/client"
	"agendago/internal/domain/passwordreset"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/session"
	"agendago/internal/pkg/token"
	ucauth "agendago/internal/usecase/auth"
)

// ambienteRecuperacao monta o par de usecases de recuperação de senha sobre
// repositórios em memória, com um prestador e um cliente com conta.
type ambienteRecuperacao struct {
	solicitar *ucauth.SolicitarRecuperacaoUseCase
	redefinir *ucauth.RedefinirSenhaUseCase
	login     *ucauth.LoginProviderUseCase
	providers *repository.ProviderMemoria
	clients   *repository.ClientMemoria
	sessoes   *repository.SessionMemoria
	resets    *repository.PasswordResetMemoria
	mailer    *email.MailerMemoria
	prestador *provider.Provider
}

func novoAmbienteRecuperacao(t *testing.T) *ambienteRecuperacao {
	t.Helper()
	hasher := security.NovoHasherArgon2id()
	senhaHash, _ := hasher.Gerar("senha-antiga")

	providers := repository.NovoProviderMemoria()
	p, _ := provider.Novo("provider-1", "João Silva", "joao@email.com", "11999998888", senhaHash)
	providers.Salvar(p)

	clients := repository.NovoClientMemoria()
	comConta, _ := client.NovoComConta("client-1", "Maria Souza", "maria@email.com", senhaHash)
	clients.Salvar(comConta)
	convidado, _ := client.NovoConvidado("client-2", "Convidada", "convidada@email.com", "11999998888")
	clients.Salvar(convidado)

	resets := repository.NovoPasswordResetMemoria()
	sessoes := repository.NovoSessionMemoria()
	mailer := email.NovaMailerMemoria()
	notificador := email.NovoNotificador(mailer, "http://localhost:5173", time.UTC, email.ExecutorSincrono)

	return &ambienteRecuperacao{
		solicitar: ucauth.NovoSolicitarRecuperacaoUseCase(providers, clients, resets, notificador),
		redefinir: ucauth.NovoRedefinirSenhaUseCase(providers, clients, resets, sessoes, hasher),
		login:     ucauth.NovoLoginProviderUseCase(providers, sessoes, hasher),
		providers: providers,
		clients:   clients,
		sessoes:   sessoes,
		resets:    resets,
		mailer:    mailer,
		prestador: p,
	}
}

// tokenDoUltimoEmail extrai o token do link presente no único email
// capturado pelo mailer fake — o link é `.../redefinir-senha?token=XXX`.
func tokenDoUltimoEmail(t *testing.T, mailer *email.MailerMemoria) string {
	t.Helper()
	enviadas := mailer.Enviadas()
	if len(enviadas) == 0 {
		t.Fatal("esperava pelo menos um email enviado")
	}
	html := enviadas[len(enviadas)-1].HTML
	const marcador = "token="
	i := indexOf(html, marcador)
	if i < 0 {
		t.Fatalf("email sem link de token: %s", html)
	}
	inicio := i + len(marcador)
	fim := inicio
	for fim < len(html) && html[fim] != '"' {
		fim++
	}
	return html[inicio:fim]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestSolicitarRecuperacao(t *testing.T) {
	t.Run("email de conta existente recebe token e email", func(t *testing.T) {
		amb := novoAmbienteRecuperacao(t)

		if err := amb.solicitar.Executar("joao@email.com"); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		enviadas := amb.mailer.Enviadas()
		if len(enviadas) != 1 {
			t.Fatalf("esperava 1 email enviado, got: %d", len(enviadas))
		}
		if enviadas[0].Para != "joao@email.com" {
			t.Errorf("esperava email para joao@email.com, got: %s", enviadas[0].Para)
		}
	})

	t.Run("email inexistente não retorna erro nem envia email (anti-enumeração)", func(t *testing.T) {
		amb := novoAmbienteRecuperacao(t)

		if err := amb.solicitar.Executar("fantasma@email.com"); err != nil {
			t.Fatalf("esperava sucesso (resposta genérica), got: %v", err)
		}
		if len(amb.mailer.Enviadas()) != 0 {
			t.Errorf("esperava zero emails, got: %d", len(amb.mailer.Enviadas()))
		}
	})

	t.Run("convidado sem conta não recebe email", func(t *testing.T) {
		amb := novoAmbienteRecuperacao(t)

		if err := amb.solicitar.Executar("convidada@email.com"); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(amb.mailer.Enviadas()) != 0 {
			t.Errorf("esperava zero emails para convidado, got: %d", len(amb.mailer.Enviadas()))
		}
	})

	t.Run("novo pedido invalida o token anterior", func(t *testing.T) {
		amb := novoAmbienteRecuperacao(t)

		amb.solicitar.Executar("joao@email.com")
		tokenAntigo := tokenDoUltimoEmail(t, amb.mailer)

		amb.solicitar.Executar("joao@email.com")

		err := amb.redefinir.Executar(tokenAntigo, "senha-nova-123")
		if err != ucauth.ErrTokenRecuperacaoInvalido {
			t.Errorf("esperava ErrTokenRecuperacaoInvalido para token antigo, got: %v", err)
		}
	})
}

func TestRedefinirSenha(t *testing.T) {
	t.Run("token válido troca a senha e revoga sessões", func(t *testing.T) {
		amb := novoAmbienteRecuperacao(t)

		// sessão ativa antes da redefinição
		loginOut, err := amb.login.Executar(ucauth.LoginInput{Email: "joao@email.com", Senha: "senha-antiga"})
		if err != nil {
			t.Fatalf("login de base falhou: %v", err)
		}

		amb.solicitar.Executar("joao@email.com")
		tok := tokenDoUltimoEmail(t, amb.mailer)

		if err := amb.redefinir.Executar(tok, "senha-nova-123"); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		// senha antiga não funciona mais
		if _, err := amb.login.Executar(ucauth.LoginInput{Email: "joao@email.com", Senha: "senha-antiga"}); err != ucauth.ErrCredenciaisInvalidas {
			t.Errorf("esperava ErrCredenciaisInvalidas com a senha antiga, got: %v", err)
		}
		// senha nova funciona
		if _, err := amb.login.Executar(ucauth.LoginInput{Email: "joao@email.com", Senha: "senha-nova-123"}); err != nil {
			t.Errorf("esperava login com a senha nova, got: %v", err)
		}
		// sessão anterior foi revogada
		s, _ := amb.sessoes.BuscarPorTokenHash(hashDoToken(loginOut.Token))
		if s != nil {
			t.Error("esperava sessão anterior revogada após redefinir a senha")
		}
	})

	t.Run("token é de uso único", func(t *testing.T) {
		amb := novoAmbienteRecuperacao(t)

		amb.solicitar.Executar("joao@email.com")
		tok := tokenDoUltimoEmail(t, amb.mailer)

		if err := amb.redefinir.Executar(tok, "senha-nova-123"); err != nil {
			t.Fatalf("primeira redefinição deveria funcionar, got: %v", err)
		}
		if err := amb.redefinir.Executar(tok, "outra-senha-456"); err != ucauth.ErrTokenRecuperacaoInvalido {
			t.Errorf("esperava ErrTokenRecuperacaoInvalido ao reusar o token, got: %v", err)
		}
	})

	t.Run("token inexistente retorna ErrTokenRecuperacaoInvalido", func(t *testing.T) {
		amb := novoAmbienteRecuperacao(t)

		if err := amb.redefinir.Executar("token-que-nunca-existiu", "senha-nova-123"); err != ucauth.ErrTokenRecuperacaoInvalido {
			t.Errorf("esperava ErrTokenRecuperacaoInvalido, got: %v", err)
		}
	})

	t.Run("token expirado retorna ErrTokenRecuperacaoInvalido", func(t *testing.T) {
		amb := novoAmbienteRecuperacao(t)

		tok, _ := token.Gerar()
		expirado := passwordreset.Novo(token.Hash(tok), amb.prestador.ID, session.TipoProvider, -time.Minute)
		amb.resets.Salvar(expirado)

		if err := amb.redefinir.Executar(tok, "senha-nova-123"); err != ucauth.ErrTokenRecuperacaoInvalido {
			t.Errorf("esperava ErrTokenRecuperacaoInvalido, got: %v", err)
		}
	})
}
