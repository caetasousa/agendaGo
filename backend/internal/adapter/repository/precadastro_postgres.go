package repository

import (
	"context"
	"errors"

	"agendago/internal/domain/precadastro"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PreCadastroPostgres struct {
	pool *pgxpool.Pool
}

func NovoPreCadastroPostgres(pool *pgxpool.Pool) *PreCadastroPostgres {
	return &PreCadastroPostgres{pool: pool}
}

// Salvar persiste um novo token de pré-cadastro.
func (r *PreCadastroPostgres) Salvar(p *precadastro.PreCadastro) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO pre_cadastro_tokens (token_hash, nome, email, telefone, criado_em)
		 VALUES ($1, $2, $3, $4, $5)`,
		p.TokenHash, p.Nome, p.Email, p.Telefone, p.CriadoEm,
	)
	return err
}

// BuscarPorTokenHash não apaga o registro — a leitura de pré-preenchimento
// pode acontecer várias vezes antes do submit final, que é quem de fato
// consome o token. Retorna (nil, nil) quando não existe.
func (r *PreCadastroPostgres) BuscarPorTokenHash(tokenHash string) (*precadastro.PreCadastro, error) {
	var p precadastro.PreCadastro
	err := r.pool.QueryRow(context.Background(),
		`SELECT token_hash, nome, email, telefone, criado_em
		 FROM pre_cadastro_tokens WHERE token_hash = $1`, tokenHash,
	).Scan(&p.TokenHash, &p.Nome, &p.Email, &p.Telefone, &p.CriadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Consumir apaga e devolve o pré-cadastro com o hash informado — o
// DELETE...RETURNING torna a leitura e a invalidação atômicas (uso único
// mesmo sob concorrência). Retorna (nil, nil) quando não existe.
func (r *PreCadastroPostgres) Consumir(tokenHash string) (*precadastro.PreCadastro, error) {
	var p precadastro.PreCadastro
	err := r.pool.QueryRow(context.Background(),
		`DELETE FROM pre_cadastro_tokens WHERE token_hash = $1
		 RETURNING token_hash, nome, email, telefone, criado_em`, tokenHash,
	).Scan(&p.TokenHash, &p.Nome, &p.Email, &p.Telefone, &p.CriadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
