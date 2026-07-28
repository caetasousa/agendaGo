package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/security"
	"agendago/internal/domain/client"
	"agendago/internal/domain/membro"
	"agendago/internal/domain/session"
	"agendago/internal/pkg/token"
	ucauth "agendago/internal/usecase/auth"
	"agendago/test/repository/memoria"
)

func hashDoToken(t string) string {
	return token.Hash(t)
}

func TestLoginProvider(t *testing.T) {
	hasher := security.NovoHasherArgon2id()
	senhaHash, _ := hasher.Gerar("12345678")

	novoAmbiente := func() (*ucauth.LoginProviderUseCase, *memoria.SessionMemoria) {
		usuarios, membros, providers := fakesDePrestador()
		criarPrestador(usuarios, membros, providers, "provider-1", "João Silva", "joao@email.com", "11999998888", senhaHash)
		sessoes := memoria.NovoSessionMemoria()
		return ucauth.NovoLoginProviderUseCase(usuarios, membros, providers, sessoes, hasher), sessoes
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
		if out.UserID != "provider-1" {
			t.Errorf("esperava UserID 'provider-1', got: %s", out.UserID)
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
	convidado, _ := client.NovoConvidado("client-2", "Convidado", "convidado@email.com", "11999998888")

	novoAmbiente := func() *ucauth.LoginClientUseCase {
		clients := memoria.NovoClientMemoria()
		clients.Salvar(comConta)
		clients.Salvar(convidado)
		sessoes := memoria.NovoSessionMemoria()
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
	t.Run("valida sessão ativa e devolve identidade com a agenda do vínculo", func(t *testing.T) {
		usuarios, membros, providers := fakesDePrestador()
		criarPrestador(usuarios, membros, providers, "user-1", "João Silva", "joao@email.com", "11999998888", "hash")
		sessoes := memoria.NovoSessionMemoria()
		s := session.Nova(hashDoToken("token-valido"), "user-1", session.TipoProvider, time.Hour)
		sessoes.Salvar(s)

		uc := ucauth.NovoValidarSessaoUseCase(sessoes, membros)
		id, err := uc.Executar("token-valido")
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if id.UserID != "user-1" {
			t.Errorf("esperava UserID 'user-1', got: %s", id.UserID)
		}
		if id.ProviderID != "user-1" {
			t.Errorf("esperava a agenda resolvida pelo vínculo, got: %s", id.ProviderID)
		}
		if id.Papel != membro.PapelDono {
			t.Errorf("esperava papel dono, got: %s", id.Papel)
		}
	})

	t.Run("rejeita sessão de prestador sem vínculo com agenda", func(t *testing.T) {
		_, membros, _ := fakesDePrestador()
		sessoes := memoria.NovoSessionMemoria()
		sessoes.Salvar(session.Nova(hashDoToken("token-orfao"), "sem-vinculo", session.TipoProvider, time.Hour))

		uc := ucauth.NovoValidarSessaoUseCase(sessoes, membros)
		if _, err := uc.Executar("token-orfao"); err != ucauth.ErrSessaoInvalida {
			t.Errorf("esperava ErrSessaoInvalida sem vínculo, got: %v", err)
		}
	})

	t.Run("rejeita token sem sessão correspondente", func(t *testing.T) {
		_, membros, _ := fakesDePrestador()
		sessoes := memoria.NovoSessionMemoria()
		uc := ucauth.NovoValidarSessaoUseCase(sessoes, membros)
		_, err := uc.Executar("token-desconhecido")
		if err != ucauth.ErrSessaoInvalida {
			t.Errorf("esperava ErrSessaoInvalida, got: %v", err)
		}
	})

	t.Run("rejeita sessão expirada", func(t *testing.T) {
		_, membros, _ := fakesDePrestador()
		sessoes := memoria.NovoSessionMemoria()
		s := session.Nova(hashDoToken("token-expirado"), "user-1", session.TipoProvider, -time.Hour)
		sessoes.Salvar(s)

		uc := ucauth.NovoValidarSessaoUseCase(sessoes, membros)
		_, err := uc.Executar("token-expirado")
		if err != ucauth.ErrSessaoInvalida {
			t.Errorf("esperava ErrSessaoInvalida, got: %v", err)
		}
	})
}

func TestPerfil(t *testing.T) {
	hasher := security.NovoHasherArgon2id()
	senhaHash, _ := hasher.Gerar("12345678")
	c, _ := client.NovoComConta("client-1", "Maria Silva", "maria@email.com", senhaHash)

	novoAmbiente := func() *ucauth.PerfilUseCase {
		usuarios, membros, providers := fakesDePrestador()
		criarPrestador(usuarios, membros, providers, "provider-1", "João Silva", "joao@email.com", "11999998888", senhaHash)
		clients := memoria.NovoClientMemoria()
		clients.Salvar(c)
		return ucauth.NovoPerfilUseCase(usuarios, providers, clients, memoria.NovoAdminMemoria())
	}

	t.Run("devolve perfil do prestador", func(t *testing.T) {
		uc := novoAmbiente()
		out, err := uc.Executar(ucauth.Identidade{
			UserID: "provider-1", Tipo: session.TipoProvider,
			ProviderID: "provider-1", Papel: membro.PapelDono,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		// Nome vem da agenda; email vem da conta. É a separação aparecendo na resposta.
		if out.Nome != "João Silva" || out.Email != "joao@email.com" {
			t.Errorf("esperava nome da agenda e email da conta, got: %+v", out)
		}
		if out.Tipo != "provider" {
			t.Errorf("esperava tipo 'provider', got: %s", out.Tipo)
		}
		if out.Provider == nil {
			t.Fatal("esperava o bloco da agenda preenchido para prestador")
		}
		if out.Provider.Papel != string(membro.PapelDono) {
			t.Errorf("esperava papel dono, got: %s", out.Provider.Papel)
		}
		if len(out.Provider.HorariosPadrao) != 2 {
			t.Errorf("esperava 2 blocos do expediente padrão sugerido, got: %v", out.Provider.HorariosPadrao)
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
		if out.Provider != nil {
			t.Errorf("esperava o bloco da agenda ausente para cliente, got: %+v", out.Provider)
		}
	})

	t.Run("retorna ErrSessaoInvalida para tipo desconhecido", func(t *testing.T) {
		uc := novoAmbiente()
		_, err := uc.Executar(ucauth.Identidade{UserID: "provider-1", Tipo: session.TipoUsuario("alienigena")})
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
		sessoes := memoria.NovoSessionMemoria()
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
		sessoes := memoria.NovoSessionMemoria()
		uc := ucauth.NovoLogoutUseCase(sessoes)
		if err := uc.Executar("token-nunca-existiu"); err != nil {
			t.Errorf("não esperava erro, got: %v", err)
		}
	})
}
