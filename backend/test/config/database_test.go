package config_test

import (
	"testing"

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
