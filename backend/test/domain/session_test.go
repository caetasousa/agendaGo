package domain_test

import (
	"testing"
	"time"

	"agendago/internal/domain/session"
)

func TestNovaSession(t *testing.T) {
	t.Run("cria sessão com expiração no futuro", func(t *testing.T) {
		s := session.Nova("hash-token", "user-1", session.TipoProvider, time.Hour)
		if s.UserID != "user-1" {
			t.Errorf("esperava UserID 'user-1', got: %s", s.UserID)
		}
		if s.UserType != session.TipoProvider {
			t.Errorf("esperava TipoProvider, got: %s", s.UserType)
		}
		if !s.ExpiraEm.After(time.Now()) {
			t.Error("esperava expiração no futuro")
		}
	})
}

func TestSessionExpirada(t *testing.T) {
	t.Run("não está expirada antes do TTL", func(t *testing.T) {
		s := session.Nova("hash-token", "user-1", session.TipoClient, time.Hour)
		if s.Expirada(time.Now()) {
			t.Error("não esperava sessão expirada")
		}
	})

	t.Run("está expirada após o TTL", func(t *testing.T) {
		s := session.Nova("hash-token", "user-1", session.TipoClient, time.Hour)
		if !s.Expirada(time.Now().Add(2 * time.Hour)) {
			t.Error("esperava sessão expirada")
		}
	})

	t.Run("no instante exato de ExpiraEm ainda não está expirada", func(t *testing.T) {
		// Expirada usa agora.After(ExpiraEm): no instante exato o "depois" ainda
		// não ocorreu, então a sessão só expira no instante seguinte.
		s := session.Nova("hash-token", "user-1", session.TipoClient, time.Hour)
		if s.Expirada(s.ExpiraEm) {
			t.Error("não esperava sessão expirada no instante exato de ExpiraEm")
		}
	})
}
