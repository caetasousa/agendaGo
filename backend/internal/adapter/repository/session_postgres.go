package repository

import (
	"context"
	"errors"
	"time"

	"agendago/internal/domain/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionPostgres struct {
	pool *pgxpool.Pool
}

func NovoSessionPostgres(pool *pgxpool.Pool) *SessionPostgres {
	return &SessionPostgres{pool: pool}
}

// Salvar persiste uma nova sessão.
func (r *SessionPostgres) Salvar(s *session.Session) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO sessions (token_hash, user_id, user_type, criado_em, expira_em)
		 VALUES ($1, $2, $3, $4, $5)`,
		s.TokenHash, s.UserID, string(s.UserType), s.CriadoEm, s.ExpiraEm,
	)
	return err
}

// BuscarPorTokenHash retorna (sessão, nil) quando encontra, (nil, nil) quando
// não existe sessão com o hash informado, e (nil, err) em falha real de infraestrutura.
func (r *SessionPostgres) BuscarPorTokenHash(hash string) (*session.Session, error) {
	var s session.Session
	var userType string
	err := r.pool.QueryRow(context.Background(),
		`SELECT token_hash, user_id, user_type, criado_em, expira_em
		 FROM sessions WHERE token_hash = $1`, hash,
	).Scan(&s.TokenHash, &s.UserID, &userType, &s.CriadoEm, &s.ExpiraEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.UserType = session.TipoUsuario(userType)
	return &s, nil
}

// Remover apaga a sessão com o hash informado. Não é erro remover uma sessão inexistente.
func (r *SessionPostgres) Remover(hash string) error {
	_, err := r.pool.Exec(context.Background(),
		`DELETE FROM sessions WHERE token_hash = $1`, hash,
	)
	return err
}

// RemoverDoUsuario apaga todas as sessões de um usuário — usado ao bani-lo,
// para o bloqueio valer imediatamente e não só no próximo login.
func (r *SessionPostgres) RemoverDoUsuario(userID string) error {
	_, err := r.pool.Exec(context.Background(),
		`DELETE FROM sessions WHERE user_id = $1`, userID,
	)
	return err
}

// RemoverExpiradas apaga todas as sessões cuja expira_em já passou.
func (r *SessionPostgres) RemoverExpiradas() error {
	_, err := r.pool.Exec(context.Background(),
		`DELETE FROM sessions WHERE expira_em < $1`, time.Now(),
	)
	return err
}
