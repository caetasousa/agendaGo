package usecase_test

import (
	"context"
	"testing"

	"agendago/internal/adapter/security"
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/socialidentity"
	ucauth "agendago/internal/usecase/auth"
	"agendago/test/repository/memoria"
)

// oidcFake simula o adapter Google (ProvedorOIDC): devolve uma identidade
// fixa em vez de falar com o Google de verdade.
type oidcFake struct {
	identidade *socialidentity.IdentidadeOIDC
	erro       error
}

func (f *oidcFake) URLAutorizacao(state, nonce string) string {
	return "https://accounts.google.com/o/oauth2/auth?state=" + state
}

func (f *oidcFake) TrocarCodigo(ctx context.Context, code, nonceEsperado string) (*socialidentity.IdentidadeOIDC, error) {
	if f.erro != nil {
		return nil, f.erro
	}
	return f.identidade, nil
}

// ambienteLoginSocial agrupa as dependências do usecase para os testes, todas
// em memória.
type ambienteLoginSocial struct {
	uc          *ucauth.LoginSocialUseCase
	clients     *memoria.ClientMemoria
	providers   *memoria.ProviderMemoria
	identidades *memoria.SocialIdentityMemoria
}

func novoAmbienteLoginSocial(fake *oidcFake) *ambienteLoginSocial {
	hasher := security.NovoHasherArgon2id()
	clients := memoria.NovoClientMemoria()
	providers := memoria.NovoProviderMemoria()
	identidades := memoria.NovoSocialIdentityMemoria()
	states := memoria.NovoOAuthStateMemoria()
	sessoes := memoria.NovoSessionMemoria()
	uc := ucauth.NovoLoginSocialUseCase(fake, clients, providers, clients, providers, identidades, states, sessoes, hasher)
	return &ambienteLoginSocial{uc: uc, clients: clients, providers: providers, identidades: identidades}
}

func iniciarEObterState(t *testing.T, uc *ucauth.LoginSocialUseCase, publico ucauth.PublicoLoginSocial) (string, string) {
	t.Helper()
	_, stateTexto, nonce, err := uc.Iniciar(publico)
	if err != nil {
		t.Fatalf("Iniciar: esperava sucesso, got: %v", err)
	}
	return stateTexto, nonce
}

func TestLoginSocialClient(t *testing.T) {
	t.Run("email inédito cria cliente novo sem senha e loga", func(t *testing.T) {
		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "google-sub-1", Email: "nova@email.com", EmailVerificado: true, Nome: "Nova Cliente",
		}}
		amb := novoAmbienteLoginSocial(fake)
		state, nonce := iniciarEObterState(t, amb.uc, ucauth.PublicoClient)

		out, err := amb.uc.Concluir(context.Background(), "code-qualquer", state, state, nonce)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.Nome != "Nova Cliente" {
			t.Errorf("esperava nome 'Nova Cliente', got: %s", out.Nome)
		}

		c, _ := amb.clients.BuscarPorEmail("nova@email.com")
		if c == nil {
			t.Fatal("esperava cliente criado")
		}
		if !c.TemConta() {
			t.Error("esperava TemConta()==true (senha sentinela), mesmo sem senha comunicada")
		}

		vinculo, _ := amb.identidades.BuscarPorProvedorSub(socialidentity.Google, "google-sub-1")
		if vinculo == nil || vinculo.UserID != c.ID {
			t.Error("esperava identidade social vinculada ao cliente criado")
		}
	})

	t.Run("identidade já vinculada loga direto, sem duplicar cliente", func(t *testing.T) {
		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "google-sub-2", Email: "repete@email.com", EmailVerificado: true, Nome: "Repete",
		}}
		amb := novoAmbienteLoginSocial(fake)

		state1, nonce1 := iniciarEObterState(t, amb.uc, ucauth.PublicoClient)
		primeiro, err := amb.uc.Concluir(context.Background(), "code-1", state1, state1, nonce1)
		if err != nil {
			t.Fatalf("primeiro login: esperava sucesso, got: %v", err)
		}

		state2, nonce2 := iniciarEObterState(t, amb.uc, ucauth.PublicoClient)
		segundo, err := amb.uc.Concluir(context.Background(), "code-2", state2, state2, nonce2)
		if err != nil {
			t.Fatalf("segundo login: esperava sucesso, got: %v", err)
		}

		if primeiro.UserID != segundo.UserID {
			t.Errorf("esperava mesmo UserID nos dois logins, got: %s e %s", primeiro.UserID, segundo.UserID)
		}

		todos, _ := amb.clients.Listar()
		if len(todos) != 1 {
			t.Errorf("esperava 1 cliente só, got: %d", len(todos))
		}
	})

	t.Run("email existente e verificado vincula preservando o ID", func(t *testing.T) {
		hasher := security.NovoHasherArgon2id()
		senhaHash, _ := hasher.Gerar("12345678")
		existente, _ := client.NovoComConta("client-existente", "Já Cadastrada", "existente@email.com", senhaHash)

		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "google-sub-3", Email: "existente@email.com", EmailVerificado: true, Nome: "Nome no Google",
		}}
		amb := novoAmbienteLoginSocial(fake)
		amb.clients.Salvar(existente)

		state, nonce := iniciarEObterState(t, amb.uc, ucauth.PublicoClient)
		out, err := amb.uc.Concluir(context.Background(), "code", state, state, nonce)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.UserID != existente.ID {
			t.Errorf("esperava vincular ao ID existente %s, got: %s", existente.ID, out.UserID)
		}

		vinculo, _ := amb.identidades.BuscarPorProvedorSub(socialidentity.Google, "google-sub-3")
		if vinculo == nil || vinculo.UserID != existente.ID {
			t.Error("esperava identidade vinculada ao cliente existente")
		}
	})

	t.Run("email não verificado não vincula a conta existente", func(t *testing.T) {
		hasher := security.NovoHasherArgon2id()
		senhaHash, _ := hasher.Gerar("12345678")
		existente, _ := client.NovoComConta("client-existente-2", "Já Cadastrada", "naoverificado@email.com", senhaHash)

		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "google-sub-4", Email: "naoverificado@email.com", EmailVerificado: false, Nome: "Impostor",
		}}
		amb := novoAmbienteLoginSocial(fake)
		amb.clients.Salvar(existente)

		state, nonce := iniciarEObterState(t, amb.uc, ucauth.PublicoClient)
		_, err := amb.uc.Concluir(context.Background(), "code", state, state, nonce)
		if err != ucauth.ErrEmailNaoVerificado {
			t.Errorf("esperava ErrEmailNaoVerificado, got: %v", err)
		}
	})

	t.Run("email não verificado não cria conta nova", func(t *testing.T) {
		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "google-sub-4b", Email: "inedito-nao-verificado@email.com", EmailVerificado: false, Nome: "Impostor",
		}}
		amb := novoAmbienteLoginSocial(fake)

		state, nonce := iniciarEObterState(t, amb.uc, ucauth.PublicoClient)
		_, err := amb.uc.Concluir(context.Background(), "code", state, state, nonce)
		if err != ucauth.ErrEmailNaoVerificado {
			t.Errorf("esperava ErrEmailNaoVerificado, got: %v", err)
		}

		c, _ := amb.clients.BuscarPorEmail("inedito-nao-verificado@email.com")
		if c != nil {
			t.Error("esperava nenhuma conta criada para email não verificado")
		}
	})

	t.Run("email já cadastrado como prestador rejeita login social de cliente", func(t *testing.T) {
		hasher := security.NovoHasherArgon2id()
		senhaHash, _ := hasher.Gerar("12345678")
		prestadorExistente, _ := provider.Novo("provider-existente", "Prestador", "cross@email.com", "11999998888", senhaHash)

		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "google-sub-cross-1", Email: "cross@email.com", EmailVerificado: true, Nome: "Cross",
		}}
		amb := novoAmbienteLoginSocial(fake)
		amb.providers.Salvar(prestadorExistente)

		state, nonce := iniciarEObterState(t, amb.uc, ucauth.PublicoClient)
		_, err := amb.uc.Concluir(context.Background(), "code", state, state, nonce)
		if err != ucauth.ErrEmailJaCadastradoOutroTipo {
			t.Errorf("esperava ErrEmailJaCadastradoOutroTipo, got: %v", err)
		}

		c, _ := amb.clients.BuscarPorEmail("cross@email.com")
		if c != nil {
			t.Error("esperava nenhum cliente duplicado criado para email já cadastrado como prestador")
		}
	})

	t.Run("state divergente do cookie rejeita como CSRF", func(t *testing.T) {
		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "google-sub-5", Email: "csrf@email.com", EmailVerificado: true, Nome: "CSRF",
		}}
		amb := novoAmbienteLoginSocial(fake)
		state, nonce := iniciarEObterState(t, amb.uc, ucauth.PublicoClient)

		_, err := amb.uc.Concluir(context.Background(), "code", state, "state-forjado", nonce)
		if err != ucauth.ErrStateInvalido {
			t.Errorf("esperava ErrStateInvalido, got: %v", err)
		}
	})

	t.Run("state desconhecido (não emitido) rejeita como CSRF", func(t *testing.T) {
		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "google-sub-6", Email: "forjado@email.com", EmailVerificado: true, Nome: "Forjado",
		}}
		amb := novoAmbienteLoginSocial(fake)

		_, err := amb.uc.Concluir(context.Background(), "code", "state-nunca-emitido", "state-nunca-emitido", "nonce")
		if err != ucauth.ErrStateInvalido {
			t.Errorf("esperava ErrStateInvalido, got: %v", err)
		}
	})

	t.Run("state consumido não pode ser reaproveitado", func(t *testing.T) {
		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "google-sub-7", Email: "reuso@email.com", EmailVerificado: true, Nome: "Reuso",
		}}
		amb := novoAmbienteLoginSocial(fake)
		state, nonce := iniciarEObterState(t, amb.uc, ucauth.PublicoClient)

		if _, err := amb.uc.Concluir(context.Background(), "code", state, state, nonce); err != nil {
			t.Fatalf("primeira conclusão: esperava sucesso, got: %v", err)
		}

		_, err := amb.uc.Concluir(context.Background(), "code", state, state, nonce)
		if err != ucauth.ErrStateInvalido {
			t.Errorf("esperava ErrStateInvalido no reuso do state, got: %v", err)
		}
	})

	t.Run("state emitido para provider não loga como client mesmo se o código combina", func(t *testing.T) {
		// o publico agora vem do state persistido (Iniciar), não de um
		// parâmetro externo — este teste prova que não dá pra "reinterpretar"
		// um state de provider como se fosse de client: o tipo de conta é
		// decidido pelo que foi gravado em Iniciar, não pelo chamador de Concluir.
		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "google-sub-8", Email: "tipo-fixo@email.com", EmailVerificado: true, Nome: "Tipo Fixo",
		}}
		amb := novoAmbienteLoginSocial(fake)
		state, nonce := iniciarEObterState(t, amb.uc, ucauth.PublicoProvider)

		out, err := amb.uc.Concluir(context.Background(), "code", state, state, nonce)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		if c, _ := amb.clients.BuscarPorEmail("tipo-fixo@email.com"); c != nil {
			t.Error("esperava que nenhum cliente fosse criado — o state foi emitido para provider")
		}
		p, _ := amb.providers.BuscarPorEmail("tipo-fixo@email.com")
		if p == nil || p.ID != out.UserID {
			t.Error("esperava que a conta criada fosse um prestador, conforme o state emitido em Iniciar(PublicoProvider)")
		}
	})
}

func TestLoginSocialProvider(t *testing.T) {
	t.Run("email inédito cria prestador novo com telefone pendente e loga", func(t *testing.T) {
		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "google-sub-p1", Email: "prestador@email.com", EmailVerificado: true, Nome: "Novo Prestador",
		}}
		amb := novoAmbienteLoginSocial(fake)

		state, nonce := iniciarEObterState(t, amb.uc, ucauth.PublicoProvider)
		out, err := amb.uc.Concluir(context.Background(), "code", state, state, nonce)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}

		p, _ := amb.providers.BuscarPorEmail("prestador@email.com")
		if p == nil {
			t.Fatal("esperava prestador criado")
		}
		if out.UserID != p.ID {
			t.Errorf("esperava UserID %s, got: %s", p.ID, out.UserID)
		}
		if p.Telefone != ucauth.TelefonePendente {
			t.Errorf("esperava telefone pendente (%q), got: %q", ucauth.TelefonePendente, p.Telefone)
		}
		if p.AceitaAgendamentos {
			t.Error("esperava agenda desativada por padrão — prestador social ainda não confirmou o telefone")
		}

		vinculo, _ := amb.identidades.BuscarPorProvedorSub(socialidentity.Google, "google-sub-p1")
		if vinculo == nil || vinculo.UserID != p.ID {
			t.Error("esperava identidade social vinculada ao prestador criado")
		}
	})

	t.Run("prestador banido não consegue logar mesmo com identidade vinculada", func(t *testing.T) {
		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "google-sub-p2", Email: "banido@email.com", EmailVerificado: true, Nome: "Banido",
		}}
		amb := novoAmbienteLoginSocial(fake)

		state1, nonce1 := iniciarEObterState(t, amb.uc, ucauth.PublicoProvider)
		if _, err := amb.uc.Concluir(context.Background(), "code", state1, state1, nonce1); err != nil {
			t.Fatalf("primeiro login: esperava sucesso, got: %v", err)
		}

		p, _ := amb.providers.BuscarPorEmail("banido@email.com")
		p.Banir()
		amb.providers.Atualizar(p)

		state2, nonce2 := iniciarEObterState(t, amb.uc, ucauth.PublicoProvider)
		_, err := amb.uc.Concluir(context.Background(), "code", state2, state2, nonce2)
		if err != ucauth.ErrUsuarioInativo {
			t.Errorf("esperava ErrUsuarioInativo, got: %v", err)
		}
	})

	t.Run("email já cadastrado como cliente rejeita login social de prestador", func(t *testing.T) {
		hasher := security.NovoHasherArgon2id()
		senhaHash, _ := hasher.Gerar("12345678")
		clienteExistente, _ := client.NovoComConta("client-existente", "Cliente", "cross2@email.com", senhaHash)

		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "google-sub-cross-2", Email: "cross2@email.com", EmailVerificado: true, Nome: "Cross",
		}}
		amb := novoAmbienteLoginSocial(fake)
		amb.clients.Salvar(clienteExistente)

		state, nonce := iniciarEObterState(t, amb.uc, ucauth.PublicoProvider)
		_, err := amb.uc.Concluir(context.Background(), "code", state, state, nonce)
		if err != ucauth.ErrEmailJaCadastradoOutroTipo {
			t.Errorf("esperava ErrEmailJaCadastradoOutroTipo, got: %v", err)
		}

		p, _ := amb.providers.BuscarPorEmail("cross2@email.com")
		if p != nil {
			t.Error("esperava nenhum prestador duplicado criado para email já cadastrado como cliente")
		}
	})
}
