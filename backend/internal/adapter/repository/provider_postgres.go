package repository

import (
	"context"
	"errors"

	"agendago/internal/domain/provider"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProviderPostgres struct {
	pool *pgxpool.Pool
}

func NovoProviderPostgres(pool *pgxpool.Pool) *ProviderPostgres {
	return &ProviderPostgres{pool: pool}
}

// Salvar persiste um novo prestador. criado_em e atualizado_em ficam a cargo
// do DEFAULT NOW() da tabela — por isso não são enviados no INSERT.
func (r *ProviderPostgres) Salvar(p *provider.Provider) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO providers (id, nome, email, senha, aceita_agendamentos, descanso_minutos)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.Nome, p.Email, p.Senha, p.AceitaAgendamentos, p.DescansoMinutos,
	)
	return err
}

// BuscarPorEmail retorna (prestador, nil) quando encontra, (nil, nil) quando
// não existe prestador com o email, e (nil, err) em falha real de infraestrutura.
func (r *ProviderPostgres) BuscarPorEmail(email string) (*provider.Provider, error) {
	var p provider.Provider
	err := r.pool.QueryRow(context.Background(),
		`SELECT id, nome, email, senha, aceita_agendamentos, descanso_minutos, criado_em, atualizado_em
		 FROM providers WHERE email = $1`, email,
	).Scan(
		&p.ID, &p.Nome, &p.Email, &p.Senha, &p.AceitaAgendamentos,
		&p.DescansoMinutos, &p.CriadoEm, &p.AtualizadoEm,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
