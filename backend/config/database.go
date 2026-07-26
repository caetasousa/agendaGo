package config

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DSNBanco monta a string de conexão do PostgreSQL a partir das variáveis
// de ambiente DB_*. sslmode=disable porque o banco local não usa TLS.
//
// A URL é montada com net/url, não por concatenação: senha forte contém
// caracteres reservados (`/`, `+`, `@`, `:`) — `openssl rand -base64 24`, que
// a doc de produção recomenda, gera isso com frequência. Sem escapar, o `/`
// encerra a autoridade da URL e a conexão falha no boot com um erro de parse
// que não parece ter nada a ver com a senha.
func DSNBanco() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD")),
		Host:     net.JoinHostPort(os.Getenv("DB_HOST"), os.Getenv("DB_PORT")),
		Path:     "/" + os.Getenv("DB_NAME"),
		RawQuery: "sslmode=disable",
	}
	return u.String()
}

// NovoPool cria um pool de conexões com o PostgreSQL e valida com um ping.
func NovoPool(ctx context.Context) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, DSNBanco())
	if err != nil {
		return nil, fmt.Errorf("criar pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping no banco: %w", err)
	}
	return pool, nil
}
