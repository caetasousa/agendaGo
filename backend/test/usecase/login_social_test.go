package usecase_test

import (
	"context"
	"testing"

	"agendago/internal/adapter/security"
	"agendago/internal/domain/client"
	"agendago/internal/domain/socialidentity"
	"agendago/internal/domain/usuario"
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
	usuarios    *memoria.UsuarioMemoria
	membros     *memoria.MembroMemoria
	providers   *memoria.ProviderMemoria
	admins      *memoria.AdminMemoria
	identidades *memoria.SocialIdentityMemoria
}

func novoAmbienteLoginSocial(fake *oidcFake) *ambienteLoginSocial {
	hasher := security.NovoHasherArgon2id()
	clients := memoria.NovoClientMemoria()
	usuarios, membros, providers := fakesDePrestador()
	admins := memoria.NovoAdminMemoria()
	identidades := memoria.NovoSocialIdentityMemoria()
	states := memoria.NovoOAuthStateMemoria()
	sessoes := memoria.NovoSessionMemoria()
	uc := ucauth.NovoLoginSocialUseCase(
		fake,
		clients, usuarios, membros, providers, admins,
		clients, usuarios, providers, membros,
		identidades, states, sessoes, hasher,
	)
	return &ambienteLoginSocial{uc: uc, clients: clients, usuarios: usuarios, membros: membros, providers: providers, admins: admins, identidades: identidades}
}

func iniciarEObterState(t *testing.T, uc *ucauth.LoginSocialUseCase, publico ucauth.PublicoLoginSocial) (string, string) {
	t.Helper()
	_, stateTexto, nonce, err := uc.Iniciar(publico)
	if err != nil {
		t.Fatalf("Iniciar: esperava sucesso, got: %v", err)
	}
	return stateTexto, nonce
}

func TestLoginSocialComEmailDoAdminEhRejeitado(t *testing.T) {
	// o mesmo email do admin, chegando por qualquer intenção (client, provider
	// ou "entrar"), não pode criar nem vincular conta — o admin é reservado.
	for _, publico := range []ucauth.PublicoLoginSocial{ucauth.PublicoClient, ucauth.PublicoProvider, ucauth.PublicoLogin} {
		t.Run(string(publico), func(t *testing.T) {
			const emailAdmin = "admin@agendago.com"
			fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
				Sub: "google-sub-admin", Email: emailAdmin, EmailVerificado: true, Nome: "Dono",
			}}
			amb := novoAmbienteLoginSocial(fake)
			seedAdmin(t, amb.admins, emailAdmin)

			state, nonce := iniciarEObterState(t, amb.uc, publico)
			_, err := amb.uc.Concluir(context.Background(), "code-qualquer", state, state, nonce)
			if err != ucauth.ErrEmailReservadoAdmin {
				t.Fatalf("esperava ErrEmailReservadoAdmin, got: %v", err)
			}

			if c, _ := amb.clients.BuscarPorEmail(emailAdmin); c != nil {
				t.Error("não deveria criar cliente com o email do admin")
			}
			if p, _ := amb.usuarios.BuscarPorEmail(emailAdmin); p != nil {
				t.Error("não deveria criar prestador com o email do admin")
			}
			if v, _ := amb.identidades.BuscarPorProvedorSub(socialidentity.Google, "google-sub-admin"); v != nil {
				t.Error("não deveria vincular identidade social ao email do admin")
			}
		})
	}
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

		todos, _, _ := amb.clients.Listar(paginaPadrao)
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
		prestadorExistente, _ := usuario.Novo("provider-existente", "cross@email.com", "11999998888", "hash")

		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "google-sub-cross-1", Email: "cross@email.com", EmailVerificado: true, Nome: "Cross",
		}}
		amb := novoAmbienteLoginSocial(fake)
		amb.usuarios.Salvar(prestadorExistente)

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
		p, _ := amb.usuarios.BuscarPorEmail("tipo-fixo@email.com")
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

		p, _ := amb.usuarios.BuscarPorEmail("prestador@email.com")
		if p == nil {
			t.Fatal("esperava prestador criado")
		}
		if out.UserID != p.ID {
			t.Errorf("esperava UserID %s, got: %s", p.ID, out.UserID)
		}
		if p.Telefone != ucauth.TelefonePendente {
			t.Errorf("esperava telefone pendente (%q), got: %q", ucauth.TelefonePendente, p.Telefone)
		}
		agenda, _ := amb.membros.BuscarPorUsuario(p.ID)
		if agenda == nil {
			t.Fatal("esperava vínculo de dono criado junto da conta")
		}
		if ag, _ := amb.providers.BuscarPorID(agenda.ProviderID); ag == nil || ag.AceitaAgendamentos {
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

		p, _ := amb.usuarios.BuscarPorEmail("banido@email.com")
		p.Banir()
		amb.usuarios.Atualizar(p)

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

		p, _ := amb.usuarios.BuscarPorEmail("cross2@email.com")
		if p != nil {
			t.Error("esperava nenhum prestador duplicado criado para email já cadastrado como cliente")
		}
	})
}

// TestLoginSocialUnificado cobre a entrada da tela de login, que não declara o
// tipo da conta: o sistema descobre sozinho, pelo vínculo social ou pelo email.
func TestLoginSocialUnificado(t *testing.T) {
	t.Run("cliente existente entra sem declarar o tipo", func(t *testing.T) {
		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "sub-uni-1", Email: "cliente@email.com", EmailVerificado: true, Nome: "Cliente Uni",
		}}
		amb := novoAmbienteLoginSocial(fake)
		hasher := security.NovoHasherArgon2id()
		senhaHash, _ := hasher.Gerar("12345678")
		c, _ := client.NovoComConta("c-uni", "Cliente Uni", "cliente@email.com", senhaHash)
		amb.clients.Salvar(c)

		state, nonce := iniciarEObterState(t, amb.uc, ucauth.PublicoLogin)
		out, err := amb.uc.Concluir(context.Background(), "code", state, state, nonce)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.UserID != "c-uni" {
			t.Errorf("esperava logar no cliente existente (c-uni), got: %s", out.UserID)
		}
	})

	t.Run("prestador existente entra sem declarar o tipo", func(t *testing.T) {
		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "sub-uni-2", Email: "prestador@email.com", EmailVerificado: true, Nome: "Prestador Uni",
		}}
		amb := novoAmbienteLoginSocial(fake)
		criarPrestador(amb.usuarios, amb.membros, amb.providers, "p-uni", "Prestador Uni", "prestador@email.com", "11999998888", "hash")

		state, nonce := iniciarEObterState(t, amb.uc, ucauth.PublicoLogin)
		out, err := amb.uc.Concluir(context.Background(), "code", state, state, nonce)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.UserID != "p-uni" {
			t.Errorf("esperava logar no prestador existente (p-uni), got: %s", out.UserID)
		}
	})

	// O ponto central da mudança: sem conta, o sistema NÃO inventa uma. Criar
	// exigiria adivinhar se a pessoa é cliente ou prestador — escolha que muda
	// o que ela pode fazer e que só o cadastro deve fazer.
	t.Run("email sem conta não cria nada e devolve ErrContaNaoEncontrada", func(t *testing.T) {
		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "sub-uni-3", Email: "ninguem@email.com", EmailVerificado: true, Nome: "Ninguém",
		}}
		amb := novoAmbienteLoginSocial(fake)

		state, nonce := iniciarEObterState(t, amb.uc, ucauth.PublicoLogin)
		if _, err := amb.uc.Concluir(context.Background(), "code", state, state, nonce); err != ucauth.ErrContaNaoEncontrada {
			t.Fatalf("esperava ErrContaNaoEncontrada, got: %v", err)
		}
		if c, _ := amb.clients.BuscarPorEmail("ninguem@email.com"); c != nil {
			t.Error("não deveria ter criado cliente")
		}
		if p, _ := amb.usuarios.BuscarPorEmail("ninguem@email.com"); p != nil {
			t.Error("não deveria ter criado prestador")
		}
	})

	t.Run("email não verificado é recusado antes de qualquer consulta", func(t *testing.T) {
		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "sub-uni-4", Email: "naoverificado@email.com", EmailVerificado: false, Nome: "Sem Verificar",
		}}
		amb := novoAmbienteLoginSocial(fake)

		state, nonce := iniciarEObterState(t, amb.uc, ucauth.PublicoLogin)
		if _, err := amb.uc.Concluir(context.Background(), "code", state, state, nonce); err != ucauth.ErrEmailNaoVerificado {
			t.Errorf("esperava ErrEmailNaoVerificado, got: %v", err)
		}
	})

	// Regressão: o tipo da sessão tem de vir do vínculo gravado, não do que a
	// URL pediu. Vinculado como cliente, entrar pela rota de prestador não
	// pode abrir uma sessão de prestador.
	t.Run("tipo vem do vínculo, não da rota usada", func(t *testing.T) {
		fake := &oidcFake{identidade: &socialidentity.IdentidadeOIDC{
			Sub: "sub-uni-5", Email: "vinculo@email.com", EmailVerificado: true, Nome: "Vinculo",
		}}
		amb := novoAmbienteLoginSocial(fake)

		state1, nonce1 := iniciarEObterState(t, amb.uc, ucauth.PublicoClient)
		primeiro, err := amb.uc.Concluir(context.Background(), "code-1", state1, state1, nonce1)
		if err != nil {
			t.Fatalf("primeiro login: %v", err)
		}

		// agora entra pela rota de prestador com a MESMA conta Google
		state2, nonce2 := iniciarEObterState(t, amb.uc, ucauth.PublicoProvider)
		segundo, err := amb.uc.Concluir(context.Background(), "code-2", state2, state2, nonce2)
		if err != nil {
			t.Fatalf("esperava entrar como o mesmo cliente, got: %v", err)
		}
		if segundo.UserID != primeiro.UserID {
			t.Errorf("esperava o mesmo usuário do vínculo, got: %s e %s", primeiro.UserID, segundo.UserID)
		}
		if p, _ := amb.usuarios.BuscarPorEmail("vinculo@email.com"); p != nil {
			t.Error("não deveria existir prestador: o vínculo é de cliente")
		}
	})
}
