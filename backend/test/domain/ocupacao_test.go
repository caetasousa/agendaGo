package domain_test

import (
	"testing"
	"time"

	"agendago/internal/domain/availability"
	"agendago/internal/domain/ocupacao"
)

func dataDeTeste() time.Time {
	return time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
}

func TestNovaOcupacao(t *testing.T) {
	t.Run("cria compromisso com dados válidos", func(t *testing.T) {
		o, err := ocupacao.Nova("1", "provider-1", dataDeTeste(), 14*60, 15*60, "Médico", ocupacao.OrigemManual)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if o.InicioMinutos != 840 || o.FimMinutos != 900 {
			t.Errorf("esperava 840–900, got: %d–%d", o.InicioMinutos, o.FimMinutos)
		}
		if o.Origem != ocupacao.OrigemManual {
			t.Errorf("esperava origem manual, got: %s", o.Origem)
		}
	})

	t.Run("título é opcional", func(t *testing.T) {
		o, err := ocupacao.Nova("1", "provider-1", dataDeTeste(), 14*60, 15*60, "", ocupacao.OrigemManual)
		if err != nil {
			t.Fatalf("esperava sucesso sem título, got: %v", err)
		}
		if o.Titulo != "" {
			t.Errorf("esperava título vazio, got: %q", o.Titulo)
		}
	})

	t.Run("título é aparado", func(t *testing.T) {
		o, _ := ocupacao.Nova("1", "provider-1", dataDeTeste(), 14*60, 15*60, "  Médico  ", ocupacao.OrigemManual)
		if o.Titulo != "Médico" {
			t.Errorf("esperava 'Médico' sem espaços, got: %q", o.Titulo)
		}
	})

	t.Run("retorna erro quando o prestador é vazio", func(t *testing.T) {
		_, err := ocupacao.Nova("1", "", dataDeTeste(), 14*60, 15*60, "", ocupacao.OrigemManual)
		if err != ocupacao.ErrProviderObrigatorio {
			t.Errorf("esperava ErrProviderObrigatorio, got: %v", err)
		}
	})

	t.Run("retorna erro quando a origem não é reconhecida", func(t *testing.T) {
		_, err := ocupacao.Nova("1", "provider-1", dataDeTeste(), 14*60, 15*60, "", ocupacao.Origem("google"))
		if err != ocupacao.ErrOrigemInvalida {
			t.Errorf("esperava ErrOrigemInvalida, got: %v", err)
		}
	})

	t.Run("retorna erro para título acima de 120 caracteres", func(t *testing.T) {
		longo := ""
		for range 121 {
			longo += "a"
		}
		_, err := ocupacao.Nova("1", "provider-1", dataDeTeste(), 14*60, 15*60, longo, ocupacao.OrigemManual)
		if err != ocupacao.ErrTituloLongo {
			t.Errorf("esperava ErrTituloLongo, got: %v", err)
		}
	})

	// O intervalo reusa availability.NovoTimeBlock: as regras são as mesmas dos
	// blocos de expediente, e os erros devolvidos são os de lá.
	t.Run("recusa fim anterior ou igual ao início", func(t *testing.T) {
		_, err := ocupacao.Nova("1", "provider-1", dataDeTeste(), 15*60, 14*60, "", ocupacao.OrigemManual)
		if err == nil {
			t.Error("esperava erro para intervalo invertido")
		}
	})

	t.Run("recusa intervalo fora do dia", func(t *testing.T) {
		_, err := ocupacao.Nova("1", "provider-1", dataDeTeste(), 14*60, 25*60, "", ocupacao.OrigemManual)
		if err != availability.ErrForaDoDia {
			t.Errorf("esperava ErrForaDoDia, got: %v", err)
		}
	})
}

func TestOcupacaoColide(t *testing.T) {
	// 14h–15h
	o, _ := ocupacao.Nova("1", "provider-1", dataDeTeste(), 840, 900, "", ocupacao.OrigemManual)

	casos := []struct {
		nome           string
		inicio, fim    int
		deveriaColidir bool
	}{
		{"exatamente o mesmo intervalo", 840, 900, true},
		{"começa dentro", 870, 930, true},
		{"termina dentro", 810, 870, true},
		{"engloba", 780, 960, true},
		{"contido", 850, 860, true},
		{"encosta antes, sem cruzar", 780, 840, false},
		{"encosta depois, sem cruzar", 900, 960, false},
		{"bem antes", 600, 660, false},
		{"bem depois", 1000, 1060, false},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := o.Colide(c.inicio, c.fim); got != c.deveriaColidir {
				t.Errorf("esperava colisão=%v para %d–%d, got: %v", c.deveriaColidir, c.inicio, c.fim, got)
			}
		})
	}
}
