package repository

import (
	"context"
	"errors"
	"time"

	"agendago/internal/domain/signup"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SignupPostgres struct {
	pool *pgxpool.Pool
}

func NovoSignupPostgres(pool *pgxpool.Pool) *SignupPostgres {
	return &SignupPostgres{pool: pool}
}

// Salvar persiste um novo cadastro pendente.
func (r *SignupPostgres) Salvar(p *signup.Pendente) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO cadastros_pendentes (token_hash, nome, email, telefone, senha_hash, criado_em, expira_em)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.TokenHash, p.Nome, p.Email, p.Telefone, p.SenhaHash, p.CriadoEm, p.ExpiraEm,
	)
	return err
}

// Consumir apaga e devolve o cadastro pendente com o hash informado — o
// DELETE...RETURNING torna a leitura e a invalidação atômicas (uso único
// mesmo sob concorrência). Retorna (nil, nil) quando não existe.
func (r *SignupPostgres) Consumir(tokenHash string) (*signup.Pendente, error) {
	var p signup.Pendente
	err := r.pool.QueryRow(context.Background(),
		`DELETE FROM cadastros_pendentes WHERE token_hash = $1
		 RETURNING token_hash, nome, email, telefone, senha_hash, criado_em, expira_em`, tokenHash,
	).Scan(&p.TokenHash, &p.Nome, &p.Email, &p.Telefone, &p.SenhaHash, &p.CriadoEm, &p.ExpiraEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// RemoverPorEmail apaga os cadastros pendentes de um email — usado ao emitir um
// novo, para que só o pedido mais recente seja válido.
func (r *SignupPostgres) RemoverPorEmail(email string) error {
	_, err := r.pool.Exec(context.Background(),
		`DELETE FROM cadastros_pendentes WHERE email = $1`, email,
	)
	return err
}

// RemoverExpirados apaga os cadastros pendentes cuja expira_em já passou.
func (r *SignupPostgres) RemoverExpirados() error {
	_, err := r.pool.Exec(context.Background(),
		`DELETE FROM cadastros_pendentes WHERE expira_em < $1`, time.Now(),
	)
	return err
}
