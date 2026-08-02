package domain_test

import (
	"testing"

	"agendago/internal/domain/provider"
)

func TestGerarSlug(t *testing.T) {
	casos := []struct {
		nome, entrada, esperado string
	}{
		{"nome simples", "Joao Barbeiro", "joao-barbeiro"},
		{"remove acentos", "João Barbeiro", "joao-barbeiro"},
		{"cedilha e til", "Conceição Assunção", "conceicao-assuncao"},
		{"maiúsculas", "MARIA SILVA", "maria-silva"},
		{"pontuação vira hífen único", "Dr. José  —  Clínica", "dr-jose-clinica"},
		{"hífens das pontas caem", "  -Ana-  ", "ana"},
		{"números preservados", "Studio 21", "studio-21"},
		{"só símbolos vira vazio", "!@#$%", ""},
		{"vazio continua vazio", "", ""},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := provider.GerarSlug(c.entrada); got != c.esperado {
				t.Errorf("esperava %q, got: %q", c.esperado, got)
			}
		})
	}
}

func TestDefinirSlug(t *testing.T) {
	novoPrestador := func(t *testing.T) *provider.Provider {
		t.Helper()
		p, err := provider.Novo("1", "João Silva")
		if err != nil {
			t.Fatalf("criar prestador: %v", err)
		}
		return p
	}

	t.Run("prestador nasce com slug derivado do nome", func(t *testing.T) {
		p := novoPrestador(t)
		if p.Slug != "joao-silva" {
			t.Errorf("esperava 'joao-silva', got: %q", p.Slug)
		}
	})

	t.Run("aceita slug válido", func(t *testing.T) {
		p := novoPrestador(t)
		if err := p.DefinirSlug("barbearia-do-joao"); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if p.Slug != "barbearia-do-joao" {
			t.Errorf("esperava 'barbearia-do-joao', got: %q", p.Slug)
		}
	})

	t.Run("normaliza espaços e maiúsculas", func(t *testing.T) {
		p := novoPrestador(t)
		if err := p.DefinirSlug("  Barbearia-Do-Joao  "); err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if p.Slug != "barbearia-do-joao" {
			t.Errorf("esperava minúsculas sem espaços, got: %q", p.Slug)
		}
	})

	t.Run("recusa palavra reservada", func(t *testing.T) {
		for _, reservado := range []string{"admin", "painel", "api", "login", "agendar"} {
			p := novoPrestador(t)
			if err := p.DefinirSlug(reservado); err != provider.ErrSlugReservado {
				t.Errorf("esperava ErrSlugReservado para %q, got: %v", reservado, err)
			}
		}
	})

	t.Run("recusa formato inválido", func(t *testing.T) {
		invalidos := []string{
			"ab",                 // curto demais
			"-comeca-com-hifen",  // hífen na ponta
			"termina-com-hifen-", // hífen na ponta
			"com espaço",         // espaço
			"com_underline",      // caractere fora do conjunto
			"com--hifen-duplo",   // hífens repetidos
			"acentuação",         // acento
		}
		for _, s := range invalidos {
			p := novoPrestador(t)
			if err := p.DefinirSlug(s); err != provider.ErrSlugInvalido {
				t.Errorf("esperava ErrSlugInvalido para %q, got: %v", s, err)
			}
		}
	})

	t.Run("slug inválido não altera o anterior", func(t *testing.T) {
		p := novoPrestador(t)
		antes := p.Slug
		p.DefinirSlug("!!")
		if p.Slug != antes {
			t.Errorf("esperava o slug anterior preservado, got: %q", p.Slug)
		}
	})
}
