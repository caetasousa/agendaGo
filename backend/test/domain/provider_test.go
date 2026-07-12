package domain_test

import (
	"testing"

	"agendago/internal/domain/availability"
	"agendago/internal/domain/provider"
)

func TestNovo(t *testing.T) {
	t.Run("cria provider com dados válidos e agenda desativada por padrão", func(t *testing.T) {
		p, err := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if p.Nome != "João Silva" {
			t.Errorf("esperava nome 'João Silva', got: %s", p.Nome)
		}
		if p.AceitaAgendamentos {
			t.Error("agenda deve iniciar desativada")
		}
		if p.DescansoMinutos != 0 {
			t.Errorf("descanso deve iniciar em 0, got: %d", p.DescansoMinutos)
		}
	})

	t.Run("inicia com o expediente comercial sugerido (08-12, 14-18)", func(t *testing.T) {
		p, _ := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
		if len(p.HorariosPadrao) != 2 {
			t.Fatalf("esperava 2 blocos padrão, got: %d", len(p.HorariosPadrao))
		}
		if p.HorariosPadrao[0].InicioMinutos != 8*60 || p.HorariosPadrao[1].InicioMinutos != 14*60 {
			t.Errorf("esperava blocos 08-12 e 14-18, got: %v", p.HorariosPadrao)
		}
	})

	t.Run("retorna erro quando nome é vazio", func(t *testing.T) {
		_, err := provider.Novo("1", "", "joao@email.com", "12345678")
		if err != provider.ErrNomeObrigatorio {
			t.Errorf("esperava ErrNomeObrigatorio, got: %v", err)
		}
	})

	t.Run("retorna erro quando email é vazio", func(t *testing.T) {
		_, err := provider.Novo("1", "João Silva", "", "12345678")
		if err != provider.ErrEmailObrigatorio {
			t.Errorf("esperava ErrEmailObrigatorio, got: %v", err)
		}
	})

	t.Run("retorna erro quando hash de senha é vazio", func(t *testing.T) {
		_, err := provider.Novo("1", "João Silva", "joao@email.com", "")
		if err != provider.ErrSenhaObrigatoria {
			t.Errorf("esperava ErrSenhaObrigatoria, got: %v", err)
		}
	})
}

func TestAgenda(t *testing.T) {
	t.Run("ativa a agenda do provider", func(t *testing.T) {
		p, _ := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
		p.AtivarAgenda()
		if !p.AceitaAgendamentos {
			t.Error("agenda deveria estar ativa")
		}
	})

	t.Run("desativa a agenda do provider", func(t *testing.T) {
		p, _ := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
		p.AtivarAgenda()
		p.DesativarAgenda()
		if p.AceitaAgendamentos {
			t.Error("agenda deveria estar desativada")
		}
	})
}

func TestDefinirDescanso(t *testing.T) {
	t.Run("define o tempo de descanso entre atendimentos", func(t *testing.T) {
		p, _ := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
		if err := p.DefinirDescanso(15); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if p.DescansoMinutos != 15 {
			t.Errorf("esperava 15 minutos, got: %d", p.DescansoMinutos)
		}
	})

	t.Run("retorna erro quando descanso é negativo", func(t *testing.T) {
		p, _ := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
		if err := p.DefinirDescanso(-1); err != provider.ErrDescansoInvalido {
			t.Errorf("esperava ErrDescansoInvalido, got: %v", err)
		}
	})
}

func TestDefinirHorariosPadrao(t *testing.T) {
	t.Run("substitui o expediente padrão do prestador", func(t *testing.T) {
		p, _ := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
		bloco, _ := availability.NovoTimeBlock(9*60, 12*60)

		if err := p.DefinirHorariosPadrao([]availability.TimeBlock{bloco}); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(p.HorariosPadrao) != 1 || p.HorariosPadrao[0].InicioMinutos != 9*60 {
			t.Errorf("esperava 1 bloco 09-12, got: %v", p.HorariosPadrao)
		}
	})

	t.Run("aceita lista vazia (nenhum horário padrão)", func(t *testing.T) {
		p, _ := provider.Novo("1", "João Silva", "joao@email.com", "12345678")

		if err := p.DefinirHorariosPadrao(nil); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(p.HorariosPadrao) != 0 {
			t.Errorf("esperava nenhum bloco, got: %v", p.HorariosPadrao)
		}
	})

	t.Run("mescla blocos adjacentes e retorna erro para sobreposição real", func(t *testing.T) {
		p, _ := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
		manha, _ := availability.NovoTimeBlock(8*60, 12*60)
		tarde, _ := availability.NovoTimeBlock(12*60, 18*60)

		if err := p.DefinirHorariosPadrao([]availability.TimeBlock{tarde, manha}); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(p.HorariosPadrao) != 1 || p.HorariosPadrao[0].FimMinutos != 18*60 {
			t.Errorf("esperava blocos adjacentes mesclados em 08-18, got: %v", p.HorariosPadrao)
		}

		sobreposto, _ := availability.NovoTimeBlock(10*60, 14*60)
		if err := p.DefinirHorariosPadrao([]availability.TimeBlock{manha, sobreposto}); err != availability.ErrBlocosSobrepostos {
			t.Errorf("esperava ErrBlocosSobrepostos, got: %v", err)
		}
	})
}

func TestDefinirDuracaoAtendimento(t *testing.T) {
	t.Run("inicia com 60 minutos e aceita nova duração válida", func(t *testing.T) {
		p, _ := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
		if p.DuracaoAtendimentoMinutos != 60 {
			t.Errorf("esperava duração inicial de 60, got: %d", p.DuracaoAtendimentoMinutos)
		}
		if err := p.DefinirDuracaoAtendimento(45); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if p.DuracaoAtendimentoMinutos != 45 {
			t.Errorf("esperava 45, got: %d", p.DuracaoAtendimentoMinutos)
		}
	})

	t.Run("rejeita duração fora de [15, 1440]", func(t *testing.T) {
		p, _ := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
		if err := p.DefinirDuracaoAtendimento(10); err != provider.ErrDuracaoInvalida {
			t.Errorf("esperava ErrDuracaoInvalida, got: %v", err)
		}
		if err := p.DefinirDuracaoAtendimento(1500); err != provider.ErrDuracaoInvalida {
			t.Errorf("esperava ErrDuracaoInvalida, got: %v", err)
		}
	})
}

func TestBanirReativarProvider(t *testing.T) {
	p, _ := provider.Novo("1", "João Silva", "joao@email.com", "12345678")
	if !p.Ativo {
		t.Fatal("prestador deve nascer ativo")
	}
	p.Banir()
	if p.Ativo {
		t.Error("esperava prestador inativo após banir")
	}
	p.Reativar()
	if !p.Ativo {
		t.Error("esperava prestador ativo após reativar")
	}
}
