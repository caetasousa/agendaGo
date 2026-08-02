package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/domain/auditoria"
	"agendago/internal/domain/client"
	"agendago/internal/domain/session"
	uclgpd "agendago/internal/usecase/lgpd"
	"agendago/test/repository/memoria"
)

func cenarioLgpd(t *testing.T) (*uclgpd.ExportarUseCase, *uclgpd.AnonimizarUseCase, *memoria.ClientMemoria, *memoria.AppointmentMemoria, *memoria.SessionMemoria, *memoria.AuditoriaMemoria) {
	t.Helper()
	clients := memoria.NovoClientMemoria()
	agendamentos := memoria.NovoAppointmentMemoria()
	sessoes := memoria.NovoSessionMemoria()
	resets := memoria.NovoPasswordResetMemoria()
	identidades := memoria.NovoSocialIdentityMemoria()
	trilha := memoria.NovoAuditoriaMemoria()

	return uclgpd.NovoExportarUseCase(clients, agendamentos, trilha),
		uclgpd.NovoAnonimizarUseCase(clients, sessoes, resets, identidades, trilha),
		clients, agendamentos, sessoes, trilha
}

func TestAnonimizarPreservaAgendamentos(t *testing.T) {
	_, anonimizar, clients, agendamentos, sessoes, trilha := cenarioLgpd(t)

	c, _ := client.NovoComConta("cliente-1", "Maria Silva", "maria@email.com", "hash")
	clients.Salvar(c)

	agora := time.Date(2027, 5, 10, 8, 0, 0, 0, time.UTC)
	data := time.Date(2027, 5, 20, 0, 0, 0, 0, time.UTC)
	a, _ := appointment.Novo("ag-1", "provider-1", c.ID, data, 9*60, 10*60, agora, 24*time.Hour)
	if err := agendamentos.SalvarSeLivre(a, agora); err != nil {
		t.Fatalf("preparar agendamento: %v", err)
	}

	sessoes.Salvar(session.Nova("hash-de-sessao", c.ID, session.TipoClient, time.Hour))

	if err := anonimizar.Executar(c.ID); err != nil {
		t.Fatalf("esperava sucesso, got: %v", err)
	}

	t.Run("dados pessoais somem", func(t *testing.T) {
		depois, _ := clients.BuscarPorID(c.ID)
		if depois.Nome != client.NomeAnonimizado {
			t.Errorf("esperava nome anonimizado, got: %q", depois.Nome)
		}
		if depois.Email == "maria@email.com" {
			t.Error("esperava o email original apagado")
		}
		if depois.Telefone != "" || depois.SenhaHash != "" {
			t.Error("esperava telefone e senha apagados")
		}
		if depois.Ativo {
			t.Error("esperava a conta inativa")
		}
	})

	// O ponto central da fase: appointments.client_id tem ON DELETE CASCADE, e
	// um DELETE destruiria o histórico do PRESTADOR — dado de outra pessoa.
	t.Run("o agendamento continua na agenda do prestador", func(t *testing.T) {
		ocupantes, err := agendamentos.ListarOcupantesPorPeriodo("provider-1", data, data, agora)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(ocupantes) != 1 {
			t.Fatalf("esperava o agendamento preservado, got: %d", len(ocupantes))
		}
		if ocupantes[0].InicioMinutos != 9*60 {
			t.Errorf("esperava o horário preservado, got: %d", ocupantes[0].InicioMinutos)
		}
	})

	t.Run("a sessão é revogada", func(t *testing.T) {
		s, _ := sessoes.BuscarPorTokenHash("hash-de-sessao")
		if s != nil {
			t.Error("esperava a sessão removida")
		}
	})

	t.Run("fica registrado na trilha", func(t *testing.T) {
		regs := trilha.Todos()
		if len(regs) != 1 {
			t.Fatalf("esperava 1 registro, got: %d", len(regs))
		}
		if regs[0].Acao != auditoria.AcaoAnonimizarCliente {
			t.Errorf("esperava anonimizar_cliente, got: %s", regs[0].Acao)
		}
		if regs[0].AlvoID != c.ID {
			t.Errorf("esperava alvo %s, got: %s", c.ID, regs[0].AlvoID)
		}
		// A trilha não pode recriar o que a anonimização apagou.
		if len(regs[0].Detalhe) != 0 {
			t.Errorf("esperava detalhe sem dado pessoal, got: %v", regs[0].Detalhe)
		}
	})

	t.Run("pedir de novo avisa em vez de fingir que fez", func(t *testing.T) {
		if err := anonimizar.Executar(c.ID); err != uclgpd.ErrJaAnonimizado {
			t.Errorf("esperava ErrJaAnonimizado, got: %v", err)
		}
	})
}

func TestExportarDados(t *testing.T) {
	exportar, _, clients, agendamentos, _, trilha := cenarioLgpd(t)

	c, _ := client.NovoComConta("cliente-2", "João Souza", "joao@email.com", "hash")
	clients.Salvar(c)

	agora := time.Date(2027, 5, 10, 8, 0, 0, 0, time.UTC)
	data := time.Date(2027, 5, 20, 0, 0, 0, 0, time.UTC)
	a, _ := appointment.Novo("ag-2", "provider-1", c.ID, data, 14*60, 15*60, agora, 24*time.Hour)
	agendamentos.SalvarSeLivre(a, agora)

	dados, err := exportar.Executar(c.ID)
	if err != nil {
		t.Fatalf("esperava sucesso, got: %v", err)
	}

	if dados.Nome != "João Souza" || dados.Email != "joao@email.com" {
		t.Errorf("esperava o cadastro exportado, got: %+v", dados)
	}
	if len(dados.Agendamentos) != 1 {
		t.Fatalf("esperava 1 agendamento, got: %d", len(dados.Agendamentos))
	}
	if dados.Agendamentos[0].InicioMinutos != 14*60 {
		t.Errorf("esperava o horário exportado, got: %d", dados.Agendamentos[0].InicioMinutos)
	}
	if dados.Truncado {
		t.Error("não esperava truncado com um agendamento só")
	}

	t.Run("a exportação também entra na trilha", func(t *testing.T) {
		regs := trilha.Todos()
		if len(regs) != 1 || regs[0].Acao != auditoria.AcaoExportarDados {
			t.Errorf("esperava exportar_dados na trilha, got: %+v", regs)
		}
	})

	t.Run("cliente inexistente devolve não encontrado", func(t *testing.T) {
		if _, err := exportar.Executar("nao-existe"); err != uclgpd.ErrClienteNaoEncontrado {
			t.Errorf("esperava ErrClienteNaoEncontrado, got: %v", err)
		}
	})
}
