package config_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"agendago/config"
)

// A doc do swagger expõe a superfície inteira da API e não pode existir em
// produção. O Caddy também devolve 404 nesse caminho, mas isso é configuração
// de infraestrutura — aqui garantimos que a própria API não serve a rota.
func TestSwaggerSoForaDeProducao(t *testing.T) {
	requisitar := func(t *testing.T) int {
		t.Helper()
		r := config.NovoRouter()
		req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		return resp.Code
	}

	t.Run("em produção a rota não existe", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		if got := requisitar(t); got != http.StatusNotFound {
			t.Errorf("esperava 404 em production, got: %d", got)
		}
	})

	t.Run("em desenvolvimento a doc continua disponível", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")
		if got := requisitar(t); got == http.StatusNotFound {
			t.Error("esperava a doc montada em development, got: 404")
		}
	})
}

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

func TestRateLimits(t *testing.T) {
	t.Run("sem a variável vale o padrão de 10 por minuto", func(t *testing.T) {
		t.Setenv("RATE_LIMIT_LOGIN_POR_MINUTO", "")
		if got := config.RateLimitLoginPorMinuto(); got != 10 {
			t.Errorf("esperava padrão 10, got: %d", got)
		}
	})

	t.Run("zero desliga o limite", func(t *testing.T) {
		t.Setenv("RATE_LIMIT_CONVIDADO_POR_MINUTO", "0")
		if got := config.RateLimitConvidadoPorMinuto(); got != 0 {
			t.Errorf("esperava 0 (desligado), got: %d", got)
		}
	})

	t.Run("valor inválido ou negativo cai no padrão", func(t *testing.T) {
		t.Setenv("RATE_LIMIT_LOGIN_POR_MINUTO", "abc")
		if got := config.RateLimitLoginPorMinuto(); got != 10 {
			t.Errorf("esperava padrão 10 para valor inválido, got: %d", got)
		}
		t.Setenv("RATE_LIMIT_LOGIN_POR_MINUTO", "-5")
		if got := config.RateLimitLoginPorMinuto(); got != 10 {
			t.Errorf("esperava padrão 10 para negativo, got: %d", got)
		}
	})
}
