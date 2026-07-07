package domain_test

import (
	"testing"
	"time"

	"agendago/internal/domain/availability"
)

func TestNovoTimeBlock(t *testing.T) {
	t.Run("cria bloco válido", func(t *testing.T) {
		b, err := availability.NovoTimeBlock(8*60, 12*60)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if b.InicioMinutos != 480 || b.FimMinutos != 720 {
			t.Errorf("esperava 480-720, got: %d-%d", b.InicioMinutos, b.FimMinutos)
		}
	})

	t.Run("retorna erro quando fim não é posterior ao início", func(t *testing.T) {
		_, err := availability.NovoTimeBlock(600, 600)
		if err != availability.ErrFimAntesDoInicio {
			t.Errorf("esperava ErrFimAntesDoInicio, got: %v", err)
		}
	})

	t.Run("retorna erro quando fim é anterior ao início", func(t *testing.T) {
		_, err := availability.NovoTimeBlock(600, 500)
		if err != availability.ErrFimAntesDoInicio {
			t.Errorf("esperava ErrFimAntesDoInicio, got: %v", err)
		}
	})

	t.Run("retorna erro quando início é negativo", func(t *testing.T) {
		_, err := availability.NovoTimeBlock(-15, 60)
		if err != availability.ErrForaDoDia {
			t.Errorf("esperava ErrForaDoDia, got: %v", err)
		}
	})

	t.Run("retorna erro quando fim ultrapassa o dia", func(t *testing.T) {
		_, err := availability.NovoTimeBlock(1430, 1450)
		if err != availability.ErrForaDoDia {
			t.Errorf("esperava ErrForaDoDia, got: %v", err)
		}
	})

	t.Run("aceita fim exatamente no limite do dia (1440)", func(t *testing.T) {
		_, err := availability.NovoTimeBlock(1425, 1440)
		if err != nil {
			t.Errorf("esperava sucesso no limite do dia, got: %v", err)
		}
	})

	t.Run("retorna erro quando início não é múltiplo de 15", func(t *testing.T) {
		_, err := availability.NovoTimeBlock(485, 600)
		if err != availability.ErrGranularidadeInvalida {
			t.Errorf("esperava ErrGranularidadeInvalida, got: %v", err)
		}
	})

	t.Run("retorna erro quando fim não é múltiplo de 15", func(t *testing.T) {
		_, err := availability.NovoTimeBlock(480, 607)
		if err != availability.ErrGranularidadeInvalida {
			t.Errorf("esperava ErrGranularidadeInvalida, got: %v", err)
		}
	})
}

func TestNovaWeeklySchedule(t *testing.T) {
	t.Run("retorna erro quando providerID é vazio", func(t *testing.T) {
		_, err := availability.NovaWeeklySchedule("", nil)
		if err != availability.ErrProviderIDObrigatorio {
			t.Errorf("esperava ErrProviderIDObrigatorio, got: %v", err)
		}
	})

	t.Run("mescla blocos exatamente adjacentes", func(t *testing.T) {
		b1, _ := availability.NovoTimeBlock(8*60, 12*60)
		b2, _ := availability.NovoTimeBlock(12*60, 14*60)
		s, err := availability.NovaWeeklySchedule("provider-1", map[availability.DiaSemana][]availability.TimeBlock{
			availability.Segunda: {b1, b2},
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		blocos := s.BlocosDoDia(availability.Segunda)
		if len(blocos) != 1 {
			t.Fatalf("esperava 1 bloco mesclado, got: %d", len(blocos))
		}
		if blocos[0].InicioMinutos != 8*60 || blocos[0].FimMinutos != 14*60 {
			t.Errorf("esperava bloco mesclado 08:00-14:00, got: %d-%d", blocos[0].InicioMinutos, blocos[0].FimMinutos)
		}
	})

	t.Run("retorna erro quando blocos se sobrepõem de fato", func(t *testing.T) {
		b1, _ := availability.NovoTimeBlock(8*60, 13*60)
		b2, _ := availability.NovoTimeBlock(12*60, 14*60)
		_, err := availability.NovaWeeklySchedule("provider-1", map[availability.DiaSemana][]availability.TimeBlock{
			availability.Segunda: {b1, b2},
		})
		if err != availability.ErrBlocosSobrepostos {
			t.Errorf("esperava ErrBlocosSobrepostos, got: %v", err)
		}
	})

	t.Run("mantém blocos não relacionados separados e ordenados", func(t *testing.T) {
		manha, _ := availability.NovoTimeBlock(8*60, 12*60)
		tarde, _ := availability.NovoTimeBlock(14*60, 18*60)
		// inseridos fora de ordem
		s, err := availability.NovaWeeklySchedule("provider-1", map[availability.DiaSemana][]availability.TimeBlock{
			availability.Segunda: {tarde, manha},
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		blocos := s.BlocosDoDia(availability.Segunda)
		if len(blocos) != 2 {
			t.Fatalf("esperava 2 blocos, got: %d", len(blocos))
		}
		if blocos[0].InicioMinutos != 8*60 || blocos[1].InicioMinutos != 14*60 {
			t.Error("esperava blocos ordenados por início")
		}
	})

	t.Run("dia sem blocos configurados devolve vazio", func(t *testing.T) {
		s, _ := availability.NovaWeeklySchedule("provider-1", map[availability.DiaSemana][]availability.TimeBlock{})
		if len(s.BlocosDoDia(availability.Domingo)) != 0 {
			t.Error("esperava vazio para dia não configurado")
		}
	})
}

func TestDiaSemanaDe(t *testing.T) {
	t.Run("converte time.Time para DiaSemana", func(t *testing.T) {
		// 2026-07-06 é uma segunda-feira
		data := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
		if availability.DiaSemanaDe(data) != availability.Segunda {
			t.Errorf("esperava Segunda, got: %v", availability.DiaSemanaDe(data))
		}
	})
}

func TestNovaDateException(t *testing.T) {
	data := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)

	t.Run("cria exceção de bloqueio sem blocos", func(t *testing.T) {
		e, err := availability.NovaDateException("exc-1", "provider-1", data, availability.TipoBloqueio, nil)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(e.Blocos) != 0 {
			t.Error("esperava blocos vazios para bloqueio")
		}
	})

	t.Run("retorna erro quando bloqueio recebe blocos", func(t *testing.T) {
		b, _ := availability.NovoTimeBlock(8*60, 12*60)
		_, err := availability.NovaDateException("exc-1", "provider-1", data, availability.TipoBloqueio, []availability.TimeBlock{b})
		if err != availability.ErrBloqueioComBlocos {
			t.Errorf("esperava ErrBloqueioComBlocos, got: %v", err)
		}
	})

	t.Run("cria exceção extra com blocos", func(t *testing.T) {
		b, _ := availability.NovoTimeBlock(8*60, 12*60)
		e, err := availability.NovaDateException("exc-1", "provider-1", data, availability.TipoExtra, []availability.TimeBlock{b})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(e.Blocos) != 1 {
			t.Error("esperava 1 bloco para extra")
		}
	})

	t.Run("retorna erro quando extra não tem blocos", func(t *testing.T) {
		_, err := availability.NovaDateException("exc-1", "provider-1", data, availability.TipoExtra, nil)
		if err != availability.ErrExtraSemBlocos {
			t.Errorf("esperava ErrExtraSemBlocos, got: %v", err)
		}
	})

	t.Run("retorna erro quando tipo é inválido", func(t *testing.T) {
		_, err := availability.NovaDateException("exc-1", "provider-1", data, availability.TipoExcecao("feriado"), nil)
		if err != availability.ErrTipoInvalido {
			t.Errorf("esperava ErrTipoInvalido, got: %v", err)
		}
	})

	t.Run("retorna erro quando providerID é vazio", func(t *testing.T) {
		_, err := availability.NovaDateException("exc-1", "", data, availability.TipoBloqueio, nil)
		if err != availability.ErrProviderIDObrigatorio {
			t.Errorf("esperava ErrProviderIDObrigatorio, got: %v", err)
		}
	})

	t.Run("retorna erro quando blocos extra se sobrepõem", func(t *testing.T) {
		b1, _ := availability.NovoTimeBlock(8*60, 13*60)
		b2, _ := availability.NovoTimeBlock(12*60, 14*60)
		_, err := availability.NovaDateException("exc-1", "provider-1", data, availability.TipoExtra, []availability.TimeBlock{b1, b2})
		if err != availability.ErrBlocosSobrepostos {
			t.Errorf("esperava ErrBlocosSobrepostos, got: %v", err)
		}
	})
}
