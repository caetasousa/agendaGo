package config_test

import (
	"testing"

	"agendago/config"
)

func TestCookieSeguro(t *testing.T) {
	t.Run("retorna false sem APP_ENV", func(t *testing.T) {
		t.Setenv("APP_ENV", "")
		if config.CookieSeguro() {
			t.Error("esperava false sem APP_ENV definida")
		}
	})

	t.Run("retorna false em desenvolvimento", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")
		if config.CookieSeguro() {
			t.Error("esperava false em development")
		}
	})

	t.Run("retorna true em produção", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		if !config.CookieSeguro() {
			t.Error("esperava true em production")
		}
	})
}
