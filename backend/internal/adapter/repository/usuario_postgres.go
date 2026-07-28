package repository

import (
	"context"
	"errors"
	"time"

	"agendago/internal/domain/usuario"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const colunasUsuario = `id, email, senha_hash, telefone, ativo, criado_em, atualizado_em`

// colunasUsuarioComPrefixo é a mesma lista qualificada pelo alias `u`, para os
// SELECTs que juntam usuarios com outra tabela e teriam coluna ambígua.
const colunasUsuarioComPrefixo = `u.id, u.email, u.senha_hash, u.telefone, u.ativo, u.criado_em, u.atualizado_em`

type UsuarioPostgres struct {
	pool *pgxpool.Pool
}

func NovoUsuarioPostgres(pool *pgxpool.Pool) *UsuarioPostgres {
	return &UsuarioPostgres{pool: pool}
}

// Salvar persiste um novo usuário. criado_em e atualizado_em ficam a cargo do
// DEFAULT NOW() da tabela — por isso não são enviados no INSERT.
func (r *UsuarioPostgres) Salvar(u *usuario.Usuario) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO usuarios (id, email, senha_hash, telefone, ativo)
		 VALUES ($1, $2, $3, $4, $5)`,
		u.ID, u.Email, u.SenhaHash, u.Telefone, u.Ativo,
	)
	return err
}

// BuscarPorEmail retorna (usuário, nil) quando encontra, (nil, nil) quando não
// existe usuário com o email, e (nil, err) em falha real de infraestrutura.
func (r *UsuarioPostgres) BuscarPorEmail(email string) (*usuario.Usuario, error) {
	return r.buscar(`WHERE email = $1`, email)
}

// BuscarPorID retorna (usuário, nil) quando encontra, (nil, nil) quando não
// existe usuário com o id, e (nil, err) em falha real de infraestrutura.
func (r *UsuarioPostgres) BuscarPorID(id string) (*usuario.Usuario, error) {
	return r.buscar(`WHERE id = $1`, id)
}

func (r *UsuarioPostgres) buscar(filtro string, arg any) (*usuario.Usuario, error) {
	var u usuario.Usuario
	err := r.pool.QueryRow(context.Background(),
		`SELECT `+colunasUsuario+` FROM usuarios `+filtro, arg,
	).Scan(&u.ID, &u.Email, &u.SenhaHash, &u.Telefone, &u.Ativo, &u.CriadoEm, &u.AtualizadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Atualizar persiste os dados mutáveis do usuário (telefone e situação). Não
// altera email nem senha — a senha tem método próprio.
func (r *UsuarioPostgres) Atualizar(u *usuario.Usuario) error {
	_, err := r.pool.Exec(context.Background(),
		`UPDATE usuarios SET telefone = $2, ativo = $3, atualizado_em = $4 WHERE id = $1`,
		u.ID, u.Telefone, u.Ativo, u.AtualizadoEm,
	)
	return err
}

// AtualizarSenha persiste um novo hash de senha — usado na redefinição via
// recuperação de senha. Método dedicado para não passar pelo Atualizar
// genérico, que não toca a coluna senha_hash.
func (r *UsuarioPostgres) AtualizarSenha(id, senhaHash string) error {
	_, err := r.pool.Exec(context.Background(),
		`UPDATE usuarios SET senha_hash = $2, atualizado_em = $3 WHERE id = $1`,
		id, senhaHash, time.Now(),
	)
	return err
}
