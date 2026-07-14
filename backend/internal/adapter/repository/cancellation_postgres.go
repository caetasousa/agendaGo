package repository

import (
	"context"
	"errors"
	"time"

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
		`INSERT INTO cancelamento_tokens (token_hash, appointment_id, criado_em, expira_em)
		 VALUES ($1, $2, $3, $4)`,
		t.TokenHash, t.AppointmentID, t.CriadoEm, t.ExpiraEm,
	)
	return err
}

// BuscarPorTokenHash retorna (token, nil) quando encontra, (nil, nil) quando
// não existe token com o hash, e (nil, err) em falha real de infraestrutura.
// Não apaga na leitura: a página de cancelamento lê os detalhes antes de
// decidir cancelar — quem apaga é Remover, chamado só após o cancelamento.
func (r *CancellationPostgres) BuscarPorTokenHash(hash string) (*cancellation.Token, error) {
	var t cancellation.Token
	err := r.pool.QueryRow(context.Background(),
		`SELECT token_hash, appointment_id, criado_em, expira_em
		 FROM cancelamento_tokens WHERE token_hash = $1`, hash,
	).Scan(&t.TokenHash, &t.AppointmentID, &t.CriadoEm, &t.ExpiraEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Remover apaga o token de cancelamento — uso único, chamado depois que o
// cancelamento de fato acontece, para o mesmo token não poder ser reusado.
func (r *CancellationPostgres) Remover(hash string) error {
	_, err := r.pool.Exec(context.Background(),
		`DELETE FROM cancelamento_tokens WHERE token_hash = $1`, hash,
	)
	return err
}

// RemoverExpirados apaga os tokens de cancelamento cuja expira_em já passou.
func (r *CancellationPostgres) RemoverExpirados() error {
	_, err := r.pool.Exec(context.Background(),
		`DELETE FROM cancelamento_tokens WHERE expira_em < $1`, time.Now(),
	)
	return err
}
