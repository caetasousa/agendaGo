package domain_test

import (
	"testing"
	"time"

	"agendago/internal/domain/convite"
	"agendago/internal/domain/membro"
)

func TestNovoConvite(t *testing.T) {
	t.Run("cria convite de operador com validade", func(t *testing.T) {
		c, err := convite.Novo("hash", "recepcao@email.com", "provider-1", membro.PapelOperador, time.Hour)
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if c.Email != "recepcao@email.com" || c.ProviderID != "provider-1" {
			t.Errorf("esperava convite para recepcao@email.com na provider-1, got: %+v", c)
		}
		if c.Papel != membro.PapelOperador {
			t.Errorf("esperava papel operador, got: %s", c.Papel)
		}
		if !c.ExpiraEm.After(c.CriadoEm) {
			t.Error("esperava expiração posterior à criação")
		}
	})

	t.Run("recusa convite sem email", func(t *testing.T) {
		if _, err := convite.Novo("hash", "", "provider-1", membro.PapelOperador, time.Hour); err != convite.ErrEmailObrigatorio {
			t.Errorf("esperava ErrEmailObrigatorio, got: %v", err)
		}
	})

	t.Run("recusa convite sem agenda", func(t *testing.T) {
		if _, err := convite.Novo("hash", "a@email.com", "", membro.PapelOperador, time.Hour); err != convite.ErrProviderObrigatorio {
			t.Errorf("esperava ErrProviderObrigatorio, got: %v", err)
		}
	})

	t.Run("recusa papel desconhecido", func(t *testing.T) {
		if _, err := convite.Novo("hash", "a@email.com", "provider-1", membro.Papel("gerente"), time.Hour); err != convite.ErrPapelInvalido {
			t.Errorf("esperava ErrPapelInvalido, got: %v", err)
		}
	})

	// Uma agenda tem um dono só, definido no cadastro. Transferir a propriedade
	// é outra operação, com outras consequências.
	t.Run("recusa convidar como dono", func(t *testing.T) {
		if _, err := convite.Novo("hash", "a@email.com", "provider-1", membro.PapelDono, time.Hour); err != convite.ErrNaoConvidaDono {
			t.Errorf("esperava ErrNaoConvidaDono, got: %v", err)
		}
	})
}

func TestConviteExpirado(t *testing.T) {
	t.Run("dentro da validade não está expirado", func(t *testing.T) {
		c, _ := convite.Novo("hash", "a@email.com", "provider-1", membro.PapelOperador, time.Hour)
		if c.Expirado(time.Now()) {
			t.Error("esperava convite válido")
		}
	})

	t.Run("passada a validade está expirado", func(t *testing.T) {
		c, _ := convite.Novo("hash", "a@email.com", "provider-1", membro.PapelOperador, -time.Minute)
		if !c.Expirado(time.Now()) {
			t.Error("esperava convite expirado")
		}
	})
}
