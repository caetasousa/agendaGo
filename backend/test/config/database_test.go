package config_test

import (
	"testing"
	"time"

	"agendago/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Senha forte tem caractere reservado de URL. `openssl rand -base64 24` — o
// comando que a doc de produção manda usar — gera `/` e `+` com frequência, e
// um `/` não escapado encerra a autoridade da URL: a API não sobe.
func TestDSNBancoComSenhaDeCaracteresReservados(t *testing.T) {
	senhas := map[string]string{
		"barra":       "aB3/xY9zQw+K1mNpR7sT2u",
		"arroba":      "senha@com@arroba",
		"dois-pontos": "senha:com:dois-pontos",
		"mistura":     "+QRfSV1FT1/72eHSbkD3i7e092==",
	}

	for nome, senha := range senhas {
		t.Run(nome, func(t *testing.T) {
			t.Setenv("DB_USER", "agendago")
			t.Setenv("DB_PASSWORD", senha)
			t.Setenv("DB_HOST", "postgres")
			t.Setenv("DB_PORT", "5432")
			t.Setenv("DB_NAME", "agendago")

			cfg, err := pgxpool.ParseConfig(config.DSNBanco())
			if err != nil {
				t.Fatalf("DSN inválida para senha com caractere reservado: %v", err)
			}
			if cfg.ConnConfig.Password != senha {
				t.Errorf("senha corrompida no parse: esperava %q, got %q", senha, cfg.ConnConfig.Password)
			}
			if cfg.ConnConfig.Host != "postgres" {
				t.Errorf("host errado: %q", cfg.ConnConfig.Host)
			}
			if cfg.ConnConfig.Database != "agendago" {
				t.Errorf("banco errado: %q", cfg.ConnConfig.Database)
			}
		})
	}
}

// Os limites do pool são decisão explícita, não herança do padrão do pgx
// (max(4, núcleos) — 4 numa VPS de 1 vCPU). Ausente ou inválida, a env var cai
// no padrão do projeto; presente e válida, manda.
func TestLimitesDoPoolVemDoAmbiente(t *testing.T) {
	t.Run("padrões quando o ambiente não define nada", func(t *testing.T) {
		if max := config.MaxConexoesBanco(); max != 10 {
			t.Errorf("esperava 10 conexões máximas, got: %d", max)
		}
		if min := config.MinConexoesBanco(); min != 2 {
			t.Errorf("esperava 2 conexões mínimas, got: %d", min)
		}
		if vida := config.VidaMaximaConexaoBanco(); vida != 60*time.Minute {
			t.Errorf("esperava vida máxima de 60min, got: %v", vida)
		}
		if ocio := config.OciosidadeMaximaConexaoBanco(); ocio != 30*time.Minute {
			t.Errorf("esperava ociosidade máxima de 30min, got: %v", ocio)
		}
		if checagem := config.IntervaloChecagemPoolBanco(); checagem != 60*time.Second {
			t.Errorf("esperava checagem a cada 60s, got: %v", checagem)
		}
	})

	t.Run("valores do ambiente sobrepõem o padrão", func(t *testing.T) {
		t.Setenv("DB_MAX_CONNS", "25")
		t.Setenv("DB_MIN_CONNS", "5")
		t.Setenv("DB_MAX_CONN_LIFETIME_MIN", "15")
		t.Setenv("DB_MAX_CONN_IDLE_MIN", "5")
		t.Setenv("DB_HEALTHCHECK_SEC", "10")

		if max := config.MaxConexoesBanco(); max != 25 {
			t.Errorf("esperava 25 conexões máximas, got: %d", max)
		}
		if min := config.MinConexoesBanco(); min != 5 {
			t.Errorf("esperava 5 conexões mínimas, got: %d", min)
		}
		if vida := config.VidaMaximaConexaoBanco(); vida != 15*time.Minute {
			t.Errorf("esperava vida máxima de 15min, got: %v", vida)
		}
		if ocio := config.OciosidadeMaximaConexaoBanco(); ocio != 5*time.Minute {
			t.Errorf("esperava ociosidade máxima de 5min, got: %v", ocio)
		}
		if checagem := config.IntervaloChecagemPoolBanco(); checagem != 10*time.Second {
			t.Errorf("esperava checagem a cada 10s, got: %v", checagem)
		}
	})

	t.Run("valor inválido cai no padrão", func(t *testing.T) {
		t.Setenv("DB_MAX_CONNS", "muitas")
		t.Setenv("DB_MIN_CONNS", "-3")

		if max := config.MaxConexoesBanco(); max != 10 {
			t.Errorf("esperava o padrão 10 para valor não numérico, got: %d", max)
		}
		if min := config.MinConexoesBanco(); min != 2 {
			t.Errorf("esperava o padrão 2 para valor negativo, got: %d", min)
		}
	})
}
