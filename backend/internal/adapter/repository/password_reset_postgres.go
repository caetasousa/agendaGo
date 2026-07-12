package repository

import (
	"context"
	"errors"
	"time"

	"agendago/internal/domain/passwordreset"
	"agendago/internal/domain/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PasswordResetPostgres struct {
	pool *pgxpool.Pool
}

func NovoPasswordResetPostgres(pool *pgxpool.Pool) *PasswordResetPostgres {
	return &PasswordResetPostgres{pool: pool}
}

// Salvar persiste um novo token de recuperação.
func (r *PasswordResetPostgres) Salvar(t *passwordreset.Token) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO password_reset_tokens (token_hash, user_id, user_type, criado_em, expira_em)
		 VALUES ($1, $2, $3, $4, $5)`,
		t.TokenHash, t.UserID, string(t.UserType), t.CriadoEm, t.ExpiraEm,
	)
	return err
}

// Consumir apaga e devolve o token com o hash informado — o DELETE...RETURNING
// torna a leitura e a invalidação atômicas, garantindo uso único mesmo sob
// concorrência. Retorna (nil, nil) quando não existe token com o hash.
func (r *PasswordResetPostgres) Consumir(tokenHash string) (*passwordreset.Token, error) {
	var t passwordreset.Token
	var userType string
	err := r.pool.QueryRow(context.Background(),
		`DELETE FROM password_reset_tokens WHERE token_hash = $1
		 RETURNING token_hash, user_id, user_type, criado_em, expira_em`, tokenHash,
	).Scan(&t.TokenHash, &t.UserID, &userType, &t.CriadoEm, &t.ExpiraEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.UserType = session.TipoUsuario(userType)
	return &t, nil
}

// RemoverDoUsuario apaga todos os tokens de recuperação pendentes de um
// usuário — usado ao emitir um novo pedido, para que só o token mais recente
// seja válido.
func (r *PasswordResetPostgres) RemoverDoUsuario(userID string) error {
	_, err := r.pool.Exec(context.Background(),
		`DELETE FROM password_reset_tokens WHERE user_id = $1`, userID,
	)
	return err
}

// RemoverExpirados apaga todos os tokens cuja expira_em já passou.
func (r *PasswordResetPostgres) RemoverExpirados() error {
	_, err := r.pool.Exec(context.Background(),
		`DELETE FROM password_reset_tokens WHERE expira_em < $1`, time.Now(),
	)
	return err
}
