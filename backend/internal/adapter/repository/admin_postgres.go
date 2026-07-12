package repository

import (
	"context"
	"errors"

	"agendago/internal/domain/admin"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminPostgres struct {
	pool *pgxpool.Pool
}

func NovoAdminPostgres(pool *pgxpool.Pool) *AdminPostgres {
	return &AdminPostgres{pool: pool}
}

// Salvar persiste o admin (upsert por email): a semeadura no boot atualiza o
// hash de senha se o admin já existir, mantendo o mesmo id.
func (r *AdminPostgres) Salvar(a *admin.Admin) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO admins (id, email, senha_hash)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (email) DO UPDATE SET senha_hash = EXCLUDED.senha_hash, atualizado_em = NOW()`,
		a.ID, a.Email, a.SenhaHash,
	)
	return err
}

// BuscarPorEmail retorna (admin, nil) quando encontra, (nil, nil) quando não
// existe admin com o email, e (nil, err) em falha real de infraestrutura.
func (r *AdminPostgres) BuscarPorEmail(email string) (*admin.Admin, error) {
	return r.buscar(`SELECT id, email, senha_hash, criado_em, atualizado_em FROM admins WHERE email = $1`, email)
}

// BuscarPorID retorna (admin, nil) quando encontra, (nil, nil) quando não existe.
func (r *AdminPostgres) BuscarPorID(id string) (*admin.Admin, error) {
	return r.buscar(`SELECT id, email, senha_hash, criado_em, atualizado_em FROM admins WHERE id = $1`, id)
}

func (r *AdminPostgres) buscar(sql, arg string) (*admin.Admin, error) {
	var a admin.Admin
	err := r.pool.QueryRow(context.Background(), sql, arg).
		Scan(&a.ID, &a.Email, &a.SenhaHash, &a.CriadoEm, &a.AtualizadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}
