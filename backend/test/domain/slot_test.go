package domain_test

import (
	"testing"

	"agendago/internal/domain/availability"
	"agendago/internal/domain/slot"
)

func bloco(t *testing.T, inicio, fim int) availability.TimeBlock {
	t.Helper()
	b, err := availability.NovoTimeBlock(inicio, fim)
	if err != nil {
		t.Fatalf("bloco inválido no teste: %v", err)
	}
	return b
}

func TestSlotsLivres(t *testing.T) {
	t.Run("fatia o bloco pela duração + buffer e descarta a sobra", func(t *testing.T) {
		// 08:00–12:00, atendimento de 60min + 30min de buffer:
		// slots em 08:00, 09:30 e 11:00 (12:30 não cabe — sobra descartada)
		blocos := []availability.TimeBlock{bloco(t, 8*60, 12*60)}
		livres := slot.Livres(blocos, nil, 60, 30)
		if len(livres) != 3 {
			t.Fatalf("esperava 3 slots, got: %d (%v)", len(livres), livres)
		}
		esperados := []int{8 * 60, 9*60 + 30, 11 * 60}
		for i, e := range esperados {
			if livres[i].InicioMinutos != e || livres[i].FimMinutos != e+60 {
				t.Errorf("slot %d: esperava %d–%d, got: %d–%d", i, e, e+60, livres[i].InicioMinutos, livres[i].FimMinutos)
			}
		}
	})

	t.Run("sem buffer os slots são adjacentes", func(t *testing.T) {
		blocos := []availability.TimeBlock{bloco(t, 8*60, 10*60)}
		livres := slot.Livres(blocos, nil, 60, 0)
		if len(livres) != 2 {
			t.Fatalf("esperava 2 slots, got: %d", len(livres))
		}
	})

	t.Run("intervalo ocupado remove o slot em conflito e respeita o buffer depois dele", func(t *testing.T) {
		// 08:00–12:00, 60min + 30min de buffer; 09:30–10:30 ocupado:
		// 08:00 livre; 09:30 colide; 11:00 começa exatamente após 10:30+30min de buffer
		blocos := []availability.TimeBlock{bloco(t, 8*60, 12*60)}
		ocupados := []slot.Intervalo{{InicioMinutos: 9*60 + 30, FimMinutos: 10*60 + 30}}
		livres := slot.Livres(blocos, ocupados, 60, 30)
		if len(livres) != 2 {
			t.Fatalf("esperava 2 slots, got: %d (%v)", len(livres), livres)
		}
		if livres[0].InicioMinutos != 8*60 || livres[1].InicioMinutos != 11*60 {
			t.Errorf("esperava slots em 08:00 e 11:00, got: %v", livres)
		}
	})

	t.Run("slot que encostaria na ocupação sem respeitar o buffer é descartado", func(t *testing.T) {
		// 60min sem buffer: 08:00 e 09:00; com 09:00–10:00 ocupado sobra só 08:00 (e 10:00)
		blocos := []availability.TimeBlock{bloco(t, 8*60, 11*60)}
		ocupados := []slot.Intervalo{{InicioMinutos: 9 * 60, FimMinutos: 10 * 60}}
		livres := slot.Livres(blocos, ocupados, 60, 0)
		if len(livres) != 2 || livres[0].InicioMinutos != 8*60 || livres[1].InicioMinutos != 10*60 {
			t.Errorf("esperava 08:00 e 10:00, got: %v", livres)
		}
	})

	t.Run("bloco menor que a duração não gera slot", func(t *testing.T) {
		blocos := []availability.TimeBlock{bloco(t, 8*60, 8*60+45)}
		if livres := slot.Livres(blocos, nil, 60, 0); len(livres) != 0 {
			t.Errorf("esperava nenhum slot, got: %v", livres)
		}
	})

	t.Run("duração inválida devolve vazio", func(t *testing.T) {
		blocos := []availability.TimeBlock{bloco(t, 8*60, 12*60)}
		if livres := slot.Livres(blocos, nil, 0, 10); len(livres) != 0 {
			t.Errorf("esperava vazio para duração zero, got: %v", livres)
		}
	})
}
