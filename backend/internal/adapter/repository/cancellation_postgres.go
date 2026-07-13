package repository

import (
	"context"
	"errors"

	"agendago/internal/domain/cancellation"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CancellationPostgres struct {
	pool *pgxpool.Pool
}

func NovoCancellationPostgres(pool *pgxpool.Pool) *CancellationPostgres {
	return &CancellationPostgres{pool: pool}
}

// Salvar persiste um novo token de cancelamento.
func (r *CancellationPostgres) Salvar(t *cancellation.Token) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO cancelamento_tokens (token_hash, appointment_id, criado_em)
		 VALUES ($1, $2, $3)`,
		t.TokenHash, t.AppointmentID, t.CriadoEm,
	)
	return err
}

// BuscarPorTokenHash retorna (token, nil) quando encontra, (nil, nil) quando
// não existe token com o hash, e (nil, err) em falha real de infraestrutura.
// Diferente do token de recuperação de senha, não apaga na leitura: a página
// de cancelamento lê os detalhes e depois cancela usando o mesmo token.
func (r *CancellationPostgres) BuscarPorTokenHash(hash string) (*cancellation.Token, error) {
	var t cancellation.Token
	err := r.pool.QueryRow(context.Background(),
		`SELECT token_hash, appointment_id, criado_em
		 FROM cancelamento_tokens WHERE token_hash = $1`, hash,
	).Scan(&t.TokenHash, &t.AppointmentID, &t.CriadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}
