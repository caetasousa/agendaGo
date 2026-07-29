package usecase_test

import (
	"strings"
	"testing"
	"time"

	"agendago/internal/adapter/email"
	"agendago/internal/adapter/security"
	"agendago/internal/domain/client"
	"agendago/internal/domain/membro"
	ucauth "agendago/internal/usecase/auth"
	ucmembro "agendago/internal/usecase/membro"
	"agendago/test/repository/memoria"
)

type ambienteConvite struct {
	convidar   *ucmembro.ConvidarUseCase
	cancelar   *ucmembro.CancelarConviteUseCase
	consultar  *ucmembro.ConsultarConviteUseCase
	aceitar    *ucmembro.AceitarConviteUseCase
	listar     *ucmembro.ListarEquipeUseCase
	remover    *ucmembro.RemoverMembroUseCase
	usuarios   *memoria.UsuarioMemoria
	membros    *memoria.MembroMemoria
	providers  *memoria.ProviderMemoria
	clients    *memoria.ClientMemoria
	sessoes    *memoria.SessionMemoria
	mailer     *email.MailerMemoria
	providerID string
}

func novoAmbienteConvite(t *testing.T) *ambienteConvite {
	t.Helper()
	usuarios, membros, providers := fakesDePrestador()
	criarPrestador(usuarios, membros, providers, "provider-1", "Marina Fisio", "marina@email.com", "11999998888", senhaDeTeste)

	convites := memoria.NovoConviteMemoria()
	clients := memoria.NovoClientMemoria()
	admins := memoria.NovoAdminMemoria()
	sessoes := memoria.NovoSessionMemoria()
	mailer := email.NovaMailerMemoria()
	notificador := email.NovoNotificador(mailer, "http://localhost:5173", time.UTC, email.ExecutorSincrono)
	hasher := security.NovoHasherArgon2id()

	return &ambienteConvite{
		convidar:   ucmembro.NovoConvidarUseCase(convites, usuarios, clients, admins, providers, notificador),
		cancelar:   ucmembro.NovoCancelarConviteUseCase(convites),
		consultar:  ucmembro.NovoConsultarConviteUseCase(convites, providers),
		aceitar:    ucmembro.NovoAceitarConviteUseCase(convites, usuarios, membros, providers, hasher),
		listar:     ucmembro.NovoListarEquipeUseCase(membros, usuarios, convites),
		remover:    ucmembro.NovoRemoverMembroUseCase(membros, membros, usuarios, sessoes),
		usuarios:   usuarios,
		membros:    membros,
		providers:  providers,
		clients:    clients,
		sessoes:    sessoes,
		mailer:     mailer,
		providerID: "provider-1",
	}
}

// tokenDoConvite extrai o token do link que saiu no email.
func tokenDoConvite(t *testing.T, mailer *email.MailerMemoria) string {
	t.Helper()
	enviadas := mailer.Enviadas()
	if len(enviadas) == 0 {
		t.Fatal("esperava um email de convite enviado")
	}
	corpo := enviadas[len(enviadas)-1].HTML
	i := strings.Index(corpo, "/convite?token=")
	if i < 0 {
		t.Fatalf("esperava link de convite no corpo do email")
	}
	resto := corpo[i+len("/convite?token="):]
	fim := strings.IndexAny(resto, "\"' <")
	if fim < 0 {
		t.Fatal("não consegui delimitar o token no link")
	}
	return resto[:fim]
}

func TestConvidarMembro(t *testing.T) {
	t.Run("convida um email livre e manda o link", func(t *testing.T) {
		amb := novoAmbienteConvite(t)
		err := amb.convidar.Executar(ucmembro.ConvidarInput{
			ProviderID: amb.providerID, Email: "recepcao@email.com", Papel: membro.PapelOperador,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		enviadas := amb.mailer.Enviadas()
		if len(enviadas) != 1 || !strings.Contains(enviadas[0].Assunto, "Convite") {
			t.Errorf("esperava um email de convite, got: %+v", enviadas)
		}
		// O nome da agenda dá contexto a quem não esperava a mensagem.
		if !strings.Contains(enviadas[0].HTML, "Marina Fisio") {
			t.Error("esperava o nome da agenda no corpo do convite")
		}
	})

	// Ela já é dona da própria agenda. Um segundo vínculo pareceria funcionar e
	// não funcionaria — a resolução devolve o vínculo mais antigo.
	t.Run("recusa email que já é conta de prestador", func(t *testing.T) {
		amb := novoAmbienteConvite(t)
		err := amb.convidar.Executar(ucmembro.ConvidarInput{
			ProviderID: amb.providerID, Email: "marina@email.com", Papel: membro.PapelOperador,
		})
		if err != ucmembro.ErrEmailIndisponivel {
			t.Errorf("esperava ErrEmailIndisponivel, got: %v", err)
		}
		if len(amb.mailer.Enviadas()) != 0 {
			t.Error("não deveria enviar email para conta existente")
		}
	})

	// O email é único entre clientes e prestadores; duplicar tiraria dela o
	// acesso à própria conta de cliente no login unificado.
	t.Run("recusa email que já é conta de cliente", func(t *testing.T) {
		amb := novoAmbienteConvite(t)
		c, _ := client.NovoComConta("c-1", "Joana Cliente", "joana@email.com", "hash")
		amb.clients.Salvar(c)

		err := amb.convidar.Executar(ucmembro.ConvidarInput{
			ProviderID: amb.providerID, Email: "joana@email.com", Papel: membro.PapelOperador,
		})
		if err != ucmembro.ErrEmailIndisponivel {
			t.Errorf("esperava ErrEmailIndisponivel, got: %v", err)
		}
	})

	t.Run("reconvidar substitui o convite anterior", func(t *testing.T) {
		amb := novoAmbienteConvite(t)
		in := ucmembro.ConvidarInput{ProviderID: amb.providerID, Email: "recepcao@email.com", Papel: membro.PapelOperador}
		if err := amb.convidar.Executar(in); err != nil {
			t.Fatalf("primeiro convite: %v", err)
		}
		primeiro := tokenDoConvite(t, amb.mailer)
		if err := amb.convidar.Executar(in); err != nil {
			t.Fatalf("segundo convite: %v", err)
		}
		segundo := tokenDoConvite(t, amb.mailer)

		if primeiro == segundo {
			t.Fatal("esperava um token novo no reconvite")
		}
		if _, err := amb.consultar.Executar(primeiro); err != ucmembro.ErrConviteInvalido {
			t.Errorf("esperava o token anterior invalidado, got: %v", err)
		}
		if _, err := amb.consultar.Executar(segundo); err != nil {
			t.Errorf("esperava o token novo válido, got: %v", err)
		}
	})
}

func TestAceitarConvite(t *testing.T) {
	convidarEObterToken := func(t *testing.T, amb *ambienteConvite) string {
		t.Helper()
		if err := amb.convidar.Executar(ucmembro.ConvidarInput{
			ProviderID: amb.providerID, Email: "recepcao@email.com", Papel: membro.PapelOperador,
		}); err != nil {
			t.Fatalf("convidar: %v", err)
		}
		return tokenDoConvite(t, amb.mailer)
	}

	t.Run("consultar mostra a agenda sem gastar o convite", func(t *testing.T) {
		amb := novoAmbienteConvite(t)
		tok := convidarEObterToken(t, amb)

		out, err := amb.consultar.Executar(tok)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.NomeAgenda != "Marina Fisio" || out.Email != "recepcao@email.com" {
			t.Errorf("esperava o convite da Marina para a recepção, got: %+v", out)
		}
		// Abrir o link duas vezes antes de preencher o formulário é normal.
		if _, err := amb.consultar.Executar(tok); err != nil {
			t.Errorf("consultar não deveria gastar o convite, got: %v", err)
		}
	})

	t.Run("aceitar cria a conta com um vínculo só, e nenhuma agenda nova", func(t *testing.T) {
		amb := novoAmbienteConvite(t)
		tok := convidarEObterToken(t, amb)

		out, err := amb.aceitar.Executar(ucmembro.AceitarInput{
			Token: tok, Telefone: "11888887777", Senha: "12345678",
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.ProviderID != amb.providerID {
			t.Errorf("esperava vínculo com a agenda que convidou, got: %s", out.ProviderID)
		}

		u, _ := amb.usuarios.BuscarPorEmail("recepcao@email.com")
		if u == nil {
			t.Fatal("esperava a conta criada")
		}
		// O ponto do desenho: um vínculo só, então a resolução da agenda é
		// determinística e ela cai na agenda que a convidou.
		v, _ := amb.membros.BuscarPorUsuario(u.ID)
		if v == nil || v.ProviderID != amb.providerID {
			t.Fatalf("esperava vínculo único com a agenda convidante, got: %+v", v)
		}
		if v.Papel != membro.PapelOperador {
			t.Errorf("esperava papel operador, got: %s", v.Papel)
		}
	})

	t.Run("o convite é de uso único", func(t *testing.T) {
		amb := novoAmbienteConvite(t)
		tok := convidarEObterToken(t, amb)

		if _, err := amb.aceitar.Executar(ucmembro.AceitarInput{Token: tok, Telefone: "11888887777", Senha: "12345678"}); err != nil {
			t.Fatalf("primeiro aceite: %v", err)
		}
		_, err := amb.aceitar.Executar(ucmembro.AceitarInput{Token: tok, Telefone: "11777776666", Senha: "12345678"})
		if err != ucmembro.ErrConviteInvalido {
			t.Errorf("esperava ErrConviteInvalido no segundo aceite, got: %v", err)
		}
	})

	t.Run("token desconhecido é recusado", func(t *testing.T) {
		amb := novoAmbienteConvite(t)
		_, err := amb.aceitar.Executar(ucmembro.AceitarInput{Token: "inexistente", Telefone: "11888887777", Senha: "12345678"})
		if err != ucmembro.ErrConviteInvalido {
			t.Errorf("esperava ErrConviteInvalido, got: %v", err)
		}
	})

	// Quem entra por convite loga pelo caminho normal, com a senha que escolheu.
	t.Run("quem aceitou consegue logar e cai na agenda que a convidou", func(t *testing.T) {
		amb := novoAmbienteConvite(t)
		tok := convidarEObterToken(t, amb)
		if _, err := amb.aceitar.Executar(ucmembro.AceitarInput{Token: tok, Telefone: "11888887777", Senha: "12345678"}); err != nil {
			t.Fatalf("aceitar: %v", err)
		}

		hasher := security.NovoHasherArgon2id()
		login := ucauth.NovoLoginProviderUseCase(amb.usuarios, amb.membros, amb.providers, amb.sessoes, hasher)
		out, err := login.Executar(ucauth.LoginInput{Email: "recepcao@email.com", Senha: "12345678"})
		if err != nil {
			t.Fatalf("esperava login bem-sucedido, got: %v", err)
		}
		if out.Nome != "Marina Fisio" {
			t.Errorf("esperava o nome da agenda que a convidou, got: %s", out.Nome)
		}
	})
}

func TestEquipeDaAgenda(t *testing.T) {
	t.Run("lista o dono e os convites pendentes", func(t *testing.T) {
		amb := novoAmbienteConvite(t)
		if err := amb.convidar.Executar(ucmembro.ConvidarInput{
			ProviderID: amb.providerID, Email: "recepcao@email.com", Papel: membro.PapelOperador,
		}); err != nil {
			t.Fatalf("convidar: %v", err)
		}

		out, err := amb.listar.Executar(amb.providerID)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Membros) != 1 || !out.Membros[0].EhDono || out.Membros[0].Email != "marina@email.com" {
			t.Errorf("esperava só a dona na equipe, got: %+v", out.Membros)
		}
		if len(out.Pendentes) != 1 || out.Pendentes[0].Email != "recepcao@email.com" {
			t.Errorf("esperava um convite pendente, got: %+v", out.Pendentes)
		}
	})

	t.Run("cancelar o convite tira ele da lista e invalida o link", func(t *testing.T) {
		amb := novoAmbienteConvite(t)
		in := ucmembro.ConvidarInput{ProviderID: amb.providerID, Email: "recepcao@email.com", Papel: membro.PapelOperador}
		if err := amb.convidar.Executar(in); err != nil {
			t.Fatalf("convidar: %v", err)
		}
		tok := tokenDoConvite(t, amb.mailer)

		if err := amb.cancelar.Executar(ucmembro.CancelarInput{ProviderID: amb.providerID, Email: "recepcao@email.com"}); err != nil {
			t.Fatalf("cancelar: %v", err)
		}
		if _, err := amb.consultar.Executar(tok); err != ucmembro.ErrConviteInvalido {
			t.Errorf("esperava o link invalidado, got: %v", err)
		}
		out, _ := amb.listar.Executar(amb.providerID)
		if len(out.Pendentes) != 0 {
			t.Errorf("esperava nenhum pendente, got: %+v", out.Pendentes)
		}
	})

	t.Run("remover o membro apaga o vínculo e derruba as sessões", func(t *testing.T) {
		amb := novoAmbienteConvite(t)
		if err := amb.convidar.Executar(ucmembro.ConvidarInput{
			ProviderID: amb.providerID, Email: "recepcao@email.com", Papel: membro.PapelOperador,
		}); err != nil {
			t.Fatalf("convidar: %v", err)
		}
		if _, err := amb.aceitar.Executar(ucmembro.AceitarInput{
			Token: tokenDoConvite(t, amb.mailer), Telefone: "11888887777", Senha: "12345678",
		}); err != nil {
			t.Fatalf("aceitar: %v", err)
		}

		out, _ := amb.listar.Executar(amb.providerID)
		if len(out.Membros) != 2 {
			t.Fatalf("esperava dona + operadora, got: %d", len(out.Membros))
		}
		var idOperadora string
		for _, m := range out.Membros {
			if !m.EhDono {
				idOperadora = m.ID
			}
		}

		if err := amb.remover.Executar(ucmembro.RemoverInput{ProviderID: amb.providerID, MembroID: idOperadora}); err != nil {
			t.Fatalf("remover: %v", err)
		}
		depois, _ := amb.listar.Executar(amb.providerID)
		if len(depois.Membros) != 1 || !depois.Membros[0].EhDono {
			t.Errorf("esperava só a dona restando, got: %+v", depois.Membros)
		}
	})

	// Sem isso a pessoa vira uma conta fantasma: não consegue mais logar (o
	// login exige vínculo) e ainda ocupa o email para sempre, porque o convite
	// recusa qualquer endereço que já tenha conta. Ela nunca poderia voltar.
	t.Run("remover o último vínculo apaga a conta e libera o email", func(t *testing.T) {
		amb := novoAmbienteConvite(t)
		convidarEAceitar := func(email string) string {
			t.Helper()
			if err := amb.convidar.Executar(ucmembro.ConvidarInput{
				ProviderID: amb.providerID, Email: email, Papel: membro.PapelOperador,
			}); err != nil {
				t.Fatalf("convidar: %v", err)
			}
			out, err := amb.aceitar.Executar(ucmembro.AceitarInput{
				Token: tokenDoConvite(t, amb.mailer), Telefone: "11888887777", Senha: "12345678",
			})
			if err != nil {
				t.Fatalf("aceitar: %v", err)
			}
			return out.UsuarioID
		}

		usuarioID := convidarEAceitar("recepcao@email.com")
		equipe, _ := amb.listar.Executar(amb.providerID)
		var idVinculo string
		for _, m := range equipe.Membros {
			if !m.EhDono {
				idVinculo = m.ID
			}
		}

		if err := amb.remover.Executar(ucmembro.RemoverInput{ProviderID: amb.providerID, MembroID: idVinculo}); err != nil {
			t.Fatalf("remover: %v", err)
		}

		if u, _ := amb.usuarios.BuscarPorID(usuarioID); u != nil {
			t.Error("esperava a conta apagada junto com o último vínculo")
		}
		// O email volta a estar livre: a pessoa pode ser convidada de novo.
		if err := amb.convidar.Executar(ucmembro.ConvidarInput{
			ProviderID: amb.providerID, Email: "recepcao@email.com", Papel: membro.PapelOperador,
		}); err != nil {
			t.Errorf("esperava poder reconvidar o mesmo email, got: %v", err)
		}
	})

	// Uma agenda sem dono ficaria sem quem administre a conta.
	t.Run("o dono não pode ser removido", func(t *testing.T) {
		amb := novoAmbienteConvite(t)
		out, _ := amb.listar.Executar(amb.providerID)
		err := amb.remover.Executar(ucmembro.RemoverInput{ProviderID: amb.providerID, MembroID: out.Membros[0].ID})
		if err != ucmembro.ErrNaoRemoveDono {
			t.Errorf("esperava ErrNaoRemoveDono, got: %v", err)
		}
	})

	// Sem o ProviderID vindo da sessão, o id de um vínculo alheio removeria
	// acesso na agenda de outra pessoa.
	t.Run("não remove vínculo de outra agenda", func(t *testing.T) {
		amb := novoAmbienteConvite(t)
		err := amb.remover.Executar(ucmembro.RemoverInput{ProviderID: "outra-agenda", MembroID: "m-provider-1"})
		if err != ucmembro.ErrMembroNaoEncontrado {
			t.Errorf("esperava ErrMembroNaoEncontrado, got: %v", err)
		}
	})
}
