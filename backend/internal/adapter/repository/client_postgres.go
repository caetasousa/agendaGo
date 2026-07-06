package repository

import (
	"context"
	"errors"

	"agendago/internal/domain/client"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClientPostgres struct {
	pool *pgxpool.Pool
}

func NovoClientPostgres(pool *pgxpool.Pool) *ClientPostgres {
	return &ClientPostgres{pool: pool}
}

// Salvar persiste um novo cliente. criado_em e atualizado_em ficam a cargo
// do DEFAULT NOW() da tabela — por isso não são enviados no INSERT.
// SenhaHash vazio (cliente convidado) é gravado como NULL na coluna.
func (r *ClientPostgres) Salvar(c *client.Client) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO clients (id, nome, email, senha_hash)
		 VALUES ($1, $2, $3, $4)`,
		c.ID, c.Nome, c.Email, senhaHashOuNulo(c.SenhaHash),
	)
	return err
}

// BuscarPorEmail retorna (cliente, nil) quando encontra, (nil, nil) quando
// não existe cliente com o email, e (nil, err) em falha real de infraestrutura.
func (r *ClientPostgres) BuscarPorEmail(email string) (*client.Client, error) {
	var c client.Client
	var senhaHash *string
	err := r.pool.QueryRow(context.Background(),
		`SELECT id, nome, email, senha_hash, criado_em, atualizado_em
		 FROM clients WHERE email = $1`, email,
	).Scan(&c.ID, &c.Nome, &c.Email, &senhaHash, &c.CriadoEm, &c.AtualizadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if senhaHash != nil {
		c.SenhaHash = *senhaHash
	}
	return &c, nil
}

// BuscarPorID retorna (cliente, nil) quando encontra, (nil, nil) quando não
// existe cliente com o id, e (nil, err) em falha real de infraestrutura.
func (r *ClientPostgres) BuscarPorID(id string) (*client.Client, error) {
	var c client.Client
	var senhaHash *string
	err := r.pool.QueryRow(context.Background(),
		`SELECT id, nome, email, senha_hash, criado_em, atualizado_em
		 FROM clients WHERE id = $1`, id,
	).Scan(&c.ID, &c.Nome, &c.Email, &senhaHash, &c.CriadoEm, &c.AtualizadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if senhaHash != nil {
		c.SenhaHash = *senhaHash
	}
	return &c, nil
}

// senhaHashOuNulo converte a string vazia (cliente convidado) para NULL,
// já que a coluna senha_hash é opcional no banco.
func senhaHashOuNulo(senhaHash string) *string {
	if senhaHash == "" {
		return nil
	}
	return &senhaHash
}
