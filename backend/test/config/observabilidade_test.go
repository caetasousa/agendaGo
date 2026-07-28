package config_test

import (
	"testing"

	"agendago/config"
)

// O rastreamento de erro segue o mesmo contrato de EmailAtivo/OAuthGoogleAtivo:
// variável vazia desliga o recurso, sem exigir nenhuma outra configuração.
func TestRastreamentoErroDesligadoSemDSN(t *testing.T) {
	t.Setenv("SENTRY_DSN", "")

	if config.RastreamentoErroAtivo() {
		t.Error("esperava o rastreamento desligado com SENTRY_DSN vazio")
	}
	if err := config.IniciarRastreamentoErro(); err != nil {
		t.Errorf("iniciar com o recurso desligado não deveria falhar, got: %v", err)
	}
}

func TestRastreamentoErroLigadoComDSN(t *testing.T) {
	t.Setenv("SENTRY_DSN", "https://chave@exemplo.ingest.sentry.io/1")

	if !config.RastreamentoErroAtivo() {
		t.Error("esperava o rastreamento ligado com SENTRY_DSN preenchido")
	}
}

// DSN preenchido mas inválido aborta o boot em vez de seguir sem coletor: o
// operador pediu o recurso explicitamente, e degradar em silêncio esconderia
// justamente os erros que ele quis passar a enxergar.
func TestRastreamentoErroFalhaComDSNInvalido(t *testing.T) {
	t.Setenv("SENTRY_DSN", "isto-não-é-uma-dsn")

	if err := config.IniciarRastreamentoErro(); err == nil {
		t.Error("esperava erro ao iniciar com DSN inválida")
	}
}
