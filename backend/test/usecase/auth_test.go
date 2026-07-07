package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/session"
	"agendago/internal/pkg/token"
	ucauth "agendago/internal/usecase/auth"
)

func hashDoToken(t string) string {
	return token.Hash(t)
}

func TestLoginProvider(t *testing.T) {
	hasher := security.NovoHasherArgon2id()
	senhaHash, _ := hasher.Gerar("12345678")
	p, _ := provider.Novo("provider-1", "João Silva", "joao@email.com", senhaHash)

	novoAmbiente := func() (*ucauth.LoginProviderUseCase, *repository.SessionMemoria) {
		providers := repository.NovoProviderMemoria()
		providers.Salvar(p)
		sessoes := repository.NovoSessionMemoria()
		return ucauth.NovoLoginProviderUseCase(providers, sessoes, hasher), sessoes
	}

	t.Run("autentica com credenciais corretas e cria sessão", func(t *testing.T) {
		uc, sessoes := novoAmbiente()
		out, err := uc.Executar(ucauth.LoginInput{Email: "joao@email.com", Senha: "12345678"})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.Token == "" {
			t.Error("token não deve ser vazio")
		}
		if out.UserID != p.ID {
			t.Errorf("esperava UserID %s, got: %s", p.ID, out.UserID)
		}

		s, _ := sessoes.BuscarPorTokenHash(hashDoToken(out.Token))
		if s == nil {
			t.Error("esperava sessão persistida")
		}
	})

	t.Run("retorna ErrCredenciaisInvalidas para email inexistente", func(t *testing.T) {
		uc, _ := novoAmbiente()
		_, err := uc.Executar(ucauth.LoginInput{Email: "inexistente@email.com", Senha: "12345678"})
		if err != ucauth.ErrCredenciaisInvalidas {
			t.Errorf("esperava ErrCredenciaisInvalidas, got: %v", err)
		}
	})

	t.Run("retorna ErrCredenciaisInvalidas para senha incorreta", func(t *testing.T) {
		uc, _ := novoAmbiente()
		_, err := uc.Executar(ucauth.LoginInput{Email: "joao@email.com", Senha: "senha-errada"})
		if err != ucauth.ErrCredenciaisInvalidas {
			t.Errorf("esperava ErrCredenciaisInvalidas, got: %v", err)
		}
	})
}

func TestLoginClient(t *testing.T) {
	hasher := security.NovoHasherArgon2id()
	senhaHash, _ := hasher.Gerar("12345678")
	comConta, _ := client.NovoComConta("client-1", "Maria Silva", "maria@email.com", senhaHash)
	convidado, _ := client.NovoConvidado("client-2", "Convidado", "convidado@email.com")

	novoAmbiente := func() *ucauth.LoginClientUseCase {
		clients := repository.NovoClientMemoria()
		clients.Salvar(comConta)
		clients.Salvar(convidado)
		sessoes := repository.NovoSessionMemoria()
		return ucauth.NovoLoginClientUseCase(clients, sessoes, hasher)
	}

	t.Run("autentica cliente com conta e credenciais corretas", func(t *testing.T) {
		uc := novoAmbiente()
		out, err := uc.Executar(ucauth.LoginInput{Email: "maria@email.com", Senha: "12345678"})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.UserID != comConta.ID {
			t.Errorf("esperava UserID %s, got: %s", comConta.ID, out.UserID)
		}
	})

	t.Run("retorna ErrCredenciaisInvalidas para cliente convidado", func(t *testing.T) {
		uc := novoAmbiente()
		_, err := uc.Executar(ucauth.LoginInput{Email: "convidado@email.com", Senha: "qualquer"})
		if err != ucauth.ErrCredenciaisInvalidas {
			t.Errorf("esperava ErrCredenciaisInvalidas, got: %v", err)
		}
	})

	t.Run("retorna ErrCredenciaisInvalidas para email inexistente", func(t *testing.T) {
		uc := novoAmbiente()
		_, err := uc.Executar(ucauth.LoginInput{Email: "inexistente@email.com", Senha: "12345678"})
		if err != ucauth.ErrCredenciaisInvalidas {
			t.Errorf("esperava ErrCredenciaisInvalidas, got: %v", err)
		}
	})
}

func TestValidarSessao(t *testing.T) {
	t.Run("valida sessão ativa e devolve identidade", func(t *testing.T) {
		sessoes := repository.NovoSessionMemoria()
		s := session.Nova(hashDoToken("token-valido"), "user-1", session.TipoProvider, time.Hour)
		sessoes.Salvar(s)

		uc := ucauth.NovoValidarSessaoUseCase(sessoes)
		id, err := uc.Executar("token-valido")
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if id.UserID != "user-1" {
			t.Errorf("esperava UserID 'user-1', got: %s", id.UserID)
		}
	})

	t.Run("rejeita token sem sessão correspondente", func(t *testing.T) {
		sessoes := repository.NovoSessionMemoria()
		uc := ucauth.NovoValidarSessaoUseCase(sessoes)
		_, err := uc.Executar("token-desconhecido")
		if err != ucauth.ErrSessaoInvalida {
			t.Errorf("esperava ErrSessaoInvalida, got: %v", err)
		}
	})

	t.Run("rejeita sessão expirada", func(t *testing.T) {
		sessoes := repository.NovoSessionMemoria()
		s := session.Nova(hashDoToken("token-expirado"), "user-1", session.TipoProvider, -time.Hour)
		sessoes.Salvar(s)

		uc := ucauth.NovoValidarSessaoUseCase(sessoes)
		_, err := uc.Executar("token-expirado")
		if err != ucauth.ErrSessaoInvalida {
			t.Errorf("esperava ErrSessaoInvalida, got: %v", err)
		}
	})
}

func TestPerfil(t *testing.T) {
	hasher := security.NovoHasherArgon2id()
	senhaHash, _ := hasher.Gerar("12345678")
	p, _ := provider.Novo("provider-1", "João Silva", "joao@email.com", senhaHash)
	c, _ := client.NovoComConta("client-1", "Maria Silva", "maria@email.com", senhaHash)

	novoAmbiente := func() *ucauth.PerfilUseCase {
		providers := repository.NovoProviderMemoria()
		providers.Salvar(p)
		clients := repository.NovoClientMemoria()
		clients.Salvar(c)
		return ucauth.NovoPerfilUseCase(providers, clients)
	}

	t.Run("devolve perfil do prestador", func(t *testing.T) {
		uc := novoAmbiente()
		out, err := uc.Executar(ucauth.Identidade{UserID: p.ID, Tipo: session.TipoProvider})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.Nome != p.Nome || out.Email != p.Email {
			t.Errorf("esperava nome/email do prestador, got: %+v", out)
		}
		if out.Tipo != "provider" {
			t.Errorf("esperava tipo 'provider', got: %s", out.Tipo)
		}
		if out.AceitaAgendamentos == nil || out.DescansoMinutos == nil {
			t.Error("esperava AceitaAgendamentos e DescansoMinutos preenchidos para prestador")
		}
	})

	t.Run("devolve perfil do cliente", func(t *testing.T) {
		uc := novoAmbiente()
		out, err := uc.Executar(ucauth.Identidade{UserID: c.ID, Tipo: session.TipoClient})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.Nome != c.Nome || out.Email != c.Email {
			t.Errorf("esperava nome/email do cliente, got: %+v", out)
		}
		if out.Tipo != "client" {
			t.Errorf("esperava tipo 'client', got: %s", out.Tipo)
		}
		if out.AceitaAgendamentos != nil || out.DescansoMinutos != nil {
			t.Error("esperava AceitaAgendamentos e DescansoMinutos nil para cliente")
		}
	})

	t.Run("retorna ErrSessaoInvalida para tipo desconhecido", func(t *testing.T) {
		uc := novoAmbiente()
		_, err := uc.Executar(ucauth.Identidade{UserID: p.ID, Tipo: session.TipoUsuario("alienigena")})
		if err != ucauth.ErrSessaoInvalida {
			t.Errorf("esperava ErrSessaoInvalida, got: %v", err)
		}
	})

	t.Run("retorna ErrSessaoInvalida quando prestador não existe mais", func(t *testing.T) {
		uc := novoAmbiente()
		_, err := uc.Executar(ucauth.Identidade{UserID: "id-fantasma", Tipo: session.TipoProvider})
		if err != ucauth.ErrSessaoInvalida {
			t.Errorf("esperava ErrSessaoInvalida, got: %v", err)
		}
	})

	t.Run("retorna ErrSessaoInvalida quando cliente não existe mais", func(t *testing.T) {
		uc := novoAmbiente()
		_, err := uc.Executar(ucauth.Identidade{UserID: "id-fantasma", Tipo: session.TipoClient})
		if err != ucauth.ErrSessaoInvalida {
			t.Errorf("esperava ErrSessaoInvalida, got: %v", err)
		}
	})
}

func TestLogout(t *testing.T) {
	t.Run("remove a sessão", func(t *testing.T) {
		sessoes := repository.NovoSessionMemoria()
		s := session.Nova(hashDoToken("token-ativo"), "user-1", session.TipoProvider, time.Hour)
		sessoes.Salvar(s)

		uc := ucauth.NovoLogoutUseCase(sessoes)
		if err := uc.Executar("token-ativo"); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		encontrada, _ := sessoes.BuscarPorTokenHash(hashDoToken("token-ativo"))
		if encontrada != nil {
			t.Error("esperava sessão removida")
		}
	})

	t.Run("é idempotente para token sem sessão", func(t *testing.T) {
		sessoes := repository.NovoSessionMemoria()
		uc := ucauth.NovoLogoutUseCase(sessoes)
		if err := uc.Executar("token-nunca-existiu"); err != nil {
			t.Errorf("não esperava erro, got: %v", err)
		}
	})
}
