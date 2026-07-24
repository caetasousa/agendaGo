package repository

import (
	"context"
	"errors"
	"time"

	"agendago/internal/domain/oauthstate"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OAuthStatePostgres struct {
	pool *pgxpool.Pool
}

func NovoOAuthStatePostgres(pool *pgxpool.Pool) *OAuthStatePostgres {
	return &OAuthStatePostgres{pool: pool}
}

// Salvar persiste um novo state de fluxo OAuth.
func (r *OAuthStatePostgres) Salvar(s *oauthstate.State) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO oauth_states (state_hash, provedor, publico, criado_em, expira_em)
		 VALUES ($1, $2, $3, $4, $5)`,
		s.StateHash, s.Provedor, s.Publico, s.CriadoEm, s.ExpiraEm,
	)
	return err
}

// Consumir apaga e devolve o state com o hash informado — DELETE...RETURNING
// garante uso único mesmo sob concorrência. Retorna (nil, nil) quando não
// existe state com o hash (já consumido, expirado e limpo, ou forjado).
func (r *OAuthStatePostgres) Consumir(stateHash string) (*oauthstate.State, error) {
	var s oauthstate.State
	err := r.pool.QueryRow(context.Background(),
		`DELETE FROM oauth_states WHERE state_hash = $1
		 RETURNING state_hash, provedor, publico, criado_em, expira_em`, stateHash,
	).Scan(&s.StateHash, &s.Provedor, &s.Publico, &s.CriadoEm, &s.ExpiraEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// RemoverExpirados apaga todos os states cuja expira_em já passou.
func (r *OAuthStatePostgres) RemoverExpirados() error {
	_, err := r.pool.Exec(context.Background(),
		`DELETE FROM oauth_states WHERE expira_em < $1`, time.Now(),
	)
	return err
}
