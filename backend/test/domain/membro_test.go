package domain_test

import (
	"testing"

	"agendago/internal/domain/membro"
)

func TestNovoMembro(t *testing.T) {
	t.Run("cria vínculo de dono", func(t *testing.T) {
		m, err := membro.Novo("1", "usuario-1", "provider-1", membro.PapelDono)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if m.UsuarioID != "usuario-1" || m.ProviderID != "provider-1" {
			t.Errorf("esperava vínculo usuario-1/provider-1, got: %s/%s", m.UsuarioID, m.ProviderID)
		}
		if m.Papel != membro.PapelDono {
			t.Errorf("esperava papel dono, got: %s", m.Papel)
		}
	})

	t.Run("cria vínculo de operador", func(t *testing.T) {
		m, err := membro.Novo("1", "usuario-2", "provider-1", membro.PapelOperador)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if m.Papel != membro.PapelOperador {
			t.Errorf("esperava papel operador, got: %s", m.Papel)
		}
	})

	t.Run("retorna erro quando o usuário é vazio", func(t *testing.T) {
		_, err := membro.Novo("1", "", "provider-1", membro.PapelDono)
		if err != membro.ErrUsuarioObrigatorio {
			t.Errorf("esperava ErrUsuarioObrigatorio, got: %v", err)
		}
	})

	t.Run("retorna erro quando o prestador é vazio", func(t *testing.T) {
		_, err := membro.Novo("1", "usuario-1", "", membro.PapelDono)
		if err != membro.ErrProviderObrigatorio {
			t.Errorf("esperava ErrProviderObrigatorio, got: %v", err)
		}
	})

	t.Run("retorna erro quando o papel não é reconhecido", func(t *testing.T) {
		_, err := membro.Novo("1", "usuario-1", "provider-1", membro.Papel("gerente"))
		if err != membro.ErrPapelInvalido {
			t.Errorf("esperava ErrPapelInvalido, got: %v", err)
		}
	})

	t.Run("retorna erro quando o papel é vazio", func(t *testing.T) {
		_, err := membro.Novo("1", "usuario-1", "provider-1", membro.Papel(""))
		if err != membro.ErrPapelInvalido {
			t.Errorf("esperava ErrPapelInvalido, got: %v", err)
		}
	})
}

func TestPapelValido(t *testing.T) {
	casos := []struct {
		papel    membro.Papel
		esperado bool
	}{
		{membro.PapelDono, true},
		{membro.PapelOperador, true},
		{membro.Papel("gerente"), false},
		{membro.Papel(""), false},
		{membro.Papel("DONO"), false},
	}

	for _, c := range casos {
		t.Run(string(c.papel), func(t *testing.T) {
			if c.papel.Valido() != c.esperado {
				t.Errorf("esperava Valido()==%v para %q", c.esperado, c.papel)
			}
		})
	}
}

func TestPermissoesDoMembro(t *testing.T) {
	t.Run("os dois papéis gerenciam a agenda", func(t *testing.T) {
		dono, _ := membro.Novo("1", "usuario-1", "provider-1", membro.PapelDono)
		operador, _ := membro.Novo("2", "usuario-2", "provider-1", membro.PapelOperador)

		if !dono.PodeGerenciarAgenda() {
			t.Error("esperava que o dono pudesse gerenciar a agenda")
		}
		if !operador.PodeGerenciarAgenda() {
			t.Error("esperava que o operador pudesse gerenciar a agenda")
		}
	})

	t.Run("só o dono administra a conta", func(t *testing.T) {
		dono, _ := membro.Novo("1", "usuario-1", "provider-1", membro.PapelDono)
		operador, _ := membro.Novo("2", "usuario-2", "provider-1", membro.PapelOperador)

		if !dono.PodeAdministrarConta() {
			t.Error("esperava que o dono pudesse administrar a conta")
		}
		if operador.PodeAdministrarConta() {
			t.Error("esperava que o operador NÃO pudesse administrar a conta")
		}
	})
}
