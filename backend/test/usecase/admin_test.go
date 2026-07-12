package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/adapter/email"
	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/session"
	ucadmin "agendago/internal/usecase/admin"
	ucappointment "agendago/internal/usecase/appointment"
	ucauth "agendago/internal/usecase/auth"
	ucavailability "agendago/internal/usecase/availability"
)

func TestSemearAdmin(t *testing.T) {
	t.Run("cria o admin quando email e senha vêm preenchidos", func(t *testing.T) {
		repo := repository.NovoAdminMemoria()
		hasher := security.NovoHasherArgon2id()
		uc := ucadmin.NovoSemearUseCase(repo, hasher)

		if err := uc.Executar("admin@agendago.dev", "senha-forte"); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		a, _ := repo.BuscarPorEmail("admin@agendago.dev")
		if a == nil {
			t.Fatal("esperava admin semeado")
		}
		if ok, _ := hasher.Verificar("senha-forte", a.SenhaHash); !ok {
			t.Error("esperava a senha semeada válida")
		}
	})

	t.Run("é idempotente: rodar de novo mantém um só admin e atualiza a senha", func(t *testing.T) {
		repo := repository.NovoAdminMemoria()
		hasher := security.NovoHasherArgon2id()
		uc := ucadmin.NovoSemearUseCase(repo, hasher)

		uc.Executar("admin@agendago.dev", "senha-antiga")
		antigo, _ := repo.BuscarPorEmail("admin@agendago.dev")

		uc.Executar("admin@agendago.dev", "senha-nova")
		novo, _ := repo.BuscarPorEmail("admin@agendago.dev")

		if novo.ID != antigo.ID {
			t.Error("esperava o mesmo id do admin após re-semear")
		}
		if ok, _ := hasher.Verificar("senha-nova", novo.SenhaHash); !ok {
			t.Error("esperava a senha atualizada")
		}
	})

	t.Run("email ou senha vazios não criam admin", func(t *testing.T) {
		repo := repository.NovoAdminMemoria()
		uc := ucadmin.NovoSemearUseCase(repo, security.NovoHasherArgon2id())

		uc.Executar("", "senha")
		uc.Executar("admin@agendago.dev", "")
		if a, _ := repo.BuscarPorEmail("admin@agendago.dev"); a != nil {
			t.Error("esperava nenhum admin criado")
		}
	})
}

func TestModerar(t *testing.T) {
	type ambiente struct {
		uc        *ucadmin.ModerarUseCase
		providers *repository.ProviderMemoria
		clients   *repository.ClientMemoria
		sessoes   *repository.SessionMemoria
	}
	novoAmbiente := func() ambiente {
		providers := repository.NovoProviderMemoria()
		p, _ := provider.Novo("p-1", "João Prestador", "joao@email.com", "hash")
		p.AtivarAgenda()
		providers.Salvar(p)

		clients := repository.NovoClientMemoria()
		c, _ := client.NovoComConta("c-1", "Maria Cliente", "maria@email.com", "hash")
		clients.Salvar(c)

		sessoes := repository.NovoSessionMemoria()
		return ambiente{
			uc:        ucadmin.NovoModerarUseCase(providers, clients, sessoes),
			providers: providers,
			clients:   clients,
			sessoes:   sessoes,
		}
	}

	t.Run("lista prestadores e clientes com status de moderação", func(t *testing.T) {
		uc := novoAmbiente().uc

		ps, err := uc.ListarPrestadores()
		if err != nil || len(ps) != 1 || !ps[0].Ativo || !ps[0].AceitaAgendamentos {
			t.Errorf("esperava 1 prestador ativo que aceita agendamentos, got: %+v (%v)", ps, err)
		}

		cs, _ := uc.ListarClientes()
		if len(cs) != 1 || !cs[0].Ativo {
			t.Errorf("esperava 1 cliente ativo, got: %+v", cs)
		}
	})

	t.Run("banir e reativar prestador reflete no login e na listagem", func(t *testing.T) {
		amb := novoAmbiente()
		uc, providers := amb.uc, amb.providers
		sessionRepo := repository.NovoSessionMemoria()
		hasher := security.NovoHasherArgon2id()

		// re-cria o prestador com senha real para testar o login
		senhaHash, _ := hasher.Gerar("12345678")
		p, _ := provider.Novo("p-1", "João Prestador", "joao@email.com", senhaHash)
		p.AtivarAgenda()
		providers.Salvar(p)
		login := ucauth.NovoLoginProviderUseCase(providers, sessionRepo, hasher)

		if err := uc.BanirPrestador("p-1"); err != nil {
			t.Fatalf("esperava banir, got: %v", err)
		}
		if _, err := login.Executar(ucauth.LoginInput{Email: "joao@email.com", Senha: "12345678"}); err != ucauth.ErrUsuarioInativo {
			t.Errorf("esperava ErrUsuarioInativo no login do banido, got: %v", err)
		}
		ps, _ := uc.ListarPrestadores()
		if ps[0].Ativo {
			t.Error("esperava prestador inativo na listagem")
		}

		if err := uc.ReativarPrestador("p-1"); err != nil {
			t.Fatalf("esperava reativar, got: %v", err)
		}
		if _, err := login.Executar(ucauth.LoginInput{Email: "joao@email.com", Senha: "12345678"}); err != nil {
			t.Errorf("esperava login OK após reativar, got: %v", err)
		}
	})

	t.Run("banir cliente bloqueia o login dele", func(t *testing.T) {
		amb := novoAmbiente()
		uc, clients := amb.uc, amb.clients
		sessionRepo := repository.NovoSessionMemoria()
		hasher := security.NovoHasherArgon2id()

		senhaHash, _ := hasher.Gerar("12345678")
		c, _ := client.NovoComConta("c-1", "Maria Cliente", "maria@email.com", senhaHash)
		clients.Salvar(c)
		login := ucauth.NovoLoginClientUseCase(clients, sessionRepo, hasher)

		uc.BanirCliente("c-1")
		if _, err := login.Executar(ucauth.LoginInput{Email: "maria@email.com", Senha: "12345678"}); err != ucauth.ErrUsuarioInativo {
			t.Errorf("esperava ErrUsuarioInativo, got: %v", err)
		}

		// reativar devolve o acesso
		if err := uc.ReativarCliente("c-1"); err != nil {
			t.Fatalf("esperava reativar, got: %v", err)
		}
		if _, err := login.Executar(ucauth.LoginInput{Email: "maria@email.com", Senha: "12345678"}); err != nil {
			t.Errorf("esperava login OK após reativar, got: %v", err)
		}
	})

	t.Run("banir revoga as sessões ativas do usuário na hora", func(t *testing.T) {
		amb := novoAmbiente()

		// prestador e cliente com sessões ativas; um terceiro não é afetado
		amb.sessoes.Salvar(session.Nova(hashDoToken("tok-prestador"), "p-1", session.TipoProvider, time.Hour))
		amb.sessoes.Salvar(session.Nova(hashDoToken("tok-cliente"), "c-1", session.TipoClient, time.Hour))
		amb.sessoes.Salvar(session.Nova(hashDoToken("tok-outro"), "outro", session.TipoClient, time.Hour))

		if err := amb.uc.BanirPrestador("p-1"); err != nil {
			t.Fatalf("esperava banir prestador, got: %v", err)
		}
		if s, _ := amb.sessoes.BuscarPorTokenHash(hashDoToken("tok-prestador")); s != nil {
			t.Error("esperava sessão do prestador banido revogada")
		}

		if err := amb.uc.BanirCliente("c-1"); err != nil {
			t.Fatalf("esperava banir cliente, got: %v", err)
		}
		if s, _ := amb.sessoes.BuscarPorTokenHash(hashDoToken("tok-cliente")); s != nil {
			t.Error("esperava sessão do cliente banido revogada")
		}

		if s, _ := amb.sessoes.BuscarPorTokenHash(hashDoToken("tok-outro")); s == nil {
			t.Error("sessão de usuário não banido não deveria ser tocada")
		}
	})

	t.Run("banir usuário inexistente retorna erro", func(t *testing.T) {
		uc := novoAmbiente().uc
		if err := uc.BanirPrestador("fantasma"); err != ucadmin.ErrProviderNaoEncontrado {
			t.Errorf("esperava ErrProviderNaoEncontrado, got: %v", err)
		}
		if err := uc.BanirCliente("fantasma"); err != ucadmin.ErrClientNaoEncontrado {
			t.Errorf("esperava ErrClientNaoEncontrado, got: %v", err)
		}
	})
}

func TestDetalhar(t *testing.T) {
	// dia útil futuro e um "agora" bem antes dele
	segunda := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	agora := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)

	// novoAmbiente monta o DetalharUseCase sobre repositórios em memória, com um
	// prestador ativo e um convidado que agendou um horário com ele.
	novoAmbiente := func() (*ucadmin.DetalharUseCase, *provider.Provider, *client.Client) {
		providers := repository.NovoProviderMemoria()
		p, _ := provider.Novo("p-1", "João Prestador", "joao@email.com", "hash")
		p.AtivarAgenda()
		providers.Salvar(p)

		clients := repository.NovoClientMemoria()
		convidado, _ := client.NovoConvidado("c-1", "Convidada Silva", "convidada@email.com", "(11) 99999-8888")
		clients.Salvar(convidado)

		availabilityRepo := repository.NovoAvailabilityMemoria()
		appointments := repository.NovoAppointmentMemoria()
		resolvedor := ucavailability.NovoConsultarDisponibilidadeUseCase(availabilityRepo, providers)
		consultarSlots := ucappointment.NovoConsultarSlotsUseCase(resolvedor, appointments, providers, time.UTC)
		notificador := email.NovoNotificador(email.NovaMailerMemoria(), "http://localhost:5173", time.UTC, email.ExecutorSincrono)
		solicitar := ucappointment.NovoSolicitarUseCase(consultarSlots, appointments, clients, providers, notificador, 24*time.Hour)
		listar := ucappointment.NovoListarUseCase(appointments, providers, clients)

		if _, err := solicitar.Executar(ucappointment.SolicitarInput{
			ClientID: "c-1", ProviderID: "p-1", Data: segunda, InicioMinutos: 8 * 60, Agora: agora,
		}); err != nil {
			t.Fatalf("agendamento de base falhou: %v", err)
		}

		return ucadmin.NovoDetalharUseCase(providers, clients, listar), p, convidado
	}

	t.Run("detalha o prestador com dados cadastrais e agendamentos recebidos", func(t *testing.T) {
		uc, _, _ := novoAmbiente()

		d, err := uc.Prestador("p-1", agora)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if d.Nome != "João Prestador" || d.Email != "joao@email.com" {
			t.Errorf("esperava dados cadastrais do prestador, got: %+v", d)
		}
		if !d.Ativo || !d.AceitaAgendamentos {
			t.Error("esperava prestador ativo aceitando agendamentos")
		}
		if len(d.Agendamentos) != 1 {
			t.Fatalf("esperava 1 agendamento recebido, got: %d", len(d.Agendamentos))
		}
		// na visão do prestador o admin enxerga o contato do cliente
		a := d.Agendamentos[0]
		if a.NomeCliente != "Convidada Silva" || a.EmailCliente != "convidada@email.com" || a.TelefoneCliente != "(11) 99999-8888" {
			t.Errorf("esperava contato do cliente no detalhe, got: %+v", a)
		}
	})

	t.Run("detalha o cliente com telefone e agendamentos feitos", func(t *testing.T) {
		uc, _, _ := novoAmbiente()

		d, err := uc.Cliente("c-1", agora)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if d.Nome != "Convidada Silva" || d.Telefone != "(11) 99999-8888" {
			t.Errorf("esperava nome e telefone do cliente, got: %+v", d)
		}
		if d.TemConta {
			t.Error("convidado não tem conta")
		}
		if len(d.Agendamentos) != 1 {
			t.Fatalf("esperava 1 agendamento feito, got: %d", len(d.Agendamentos))
		}
		if d.Agendamentos[0].NomePrestador != "João Prestador" {
			t.Errorf("esperava o nome do prestador no detalhe do cliente, got: %+v", d.Agendamentos[0])
		}
	})

	t.Run("id inexistente retorna não encontrado", func(t *testing.T) {
		uc, _, _ := novoAmbiente()
		if _, err := uc.Prestador("fantasma", agora); err != ucadmin.ErrProviderNaoEncontrado {
			t.Errorf("esperava ErrProviderNaoEncontrado, got: %v", err)
		}
		if _, err := uc.Cliente("fantasma", agora); err != ucadmin.ErrClientNaoEncontrado {
			t.Errorf("esperava ErrClientNaoEncontrado, got: %v", err)
		}
	})
}
