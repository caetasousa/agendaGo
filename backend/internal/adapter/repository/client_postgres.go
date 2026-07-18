package repository

import (
	"context"
	"errors"
	"time"

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
// SenhaHash e Email vazios (convidado, possivelmente só-telefone) são gravados
// como NULL — o UNIQUE de email não colide entre NULLs.
func (r *ClientPostgres) Salvar(c *client.Client) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO clients (id, nome, email, telefone, senha_hash, ativo)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		c.ID, c.Nome, emailOuNulo(c.Email), telefoneOuNulo(c.Telefone), senhaHashOuNulo(c.SenhaHash), c.Ativo,
	)
	return err
}

// Atualizar persiste o estado mutável do cliente (por ora, só o banimento).
func (r *ClientPostgres) Atualizar(c *client.Client) error {
	_, err := r.pool.Exec(context.Background(),
		`UPDATE clients SET ativo = $2, atualizado_em = $3 WHERE id = $1`,
		c.ID, c.Ativo, c.AtualizadoEm,
	)
	return err
}

// AtualizarSenha persiste um novo hash de senha — usado na redefinição via
// recuperação de senha. Método dedicado para não passar pelo Atualizar
// genérico, que só toca o banimento.
func (r *ClientPostgres) AtualizarSenha(id, senhaHash string) error {
	_, err := r.pool.Exec(context.Background(),
		`UPDATE clients SET senha_hash = $2, atualizado_em = $3 WHERE id = $1`,
		id, senhaHash, time.Now(),
	)
	return err
}

// ConverterEmConta define senha e telefone num convidado, transformando-o em
// conta sem trocar o ID — assim os agendamentos que já apontam para esse
// cliente são preservados. Usado ao confirmar o cadastro por email.
func (r *ClientPostgres) ConverterEmConta(id, senhaHash, telefone string) error {
	_, err := r.pool.Exec(context.Background(),
		`UPDATE clients SET senha_hash = $2, telefone = $3, atualizado_em = $4 WHERE id = $1`,
		id, senhaHash, telefone, time.Now(),
	)
	return err
}

// BuscarPorEmail retorna (cliente, nil) quando encontra, (nil, nil) quando
// não existe cliente com o email, e (nil, err) em falha real de infraestrutura.
func (r *ClientPostgres) BuscarPorEmail(email string) (*client.Client, error) {
	return r.buscar(`SELECT id, nome, email, telefone, senha_hash, ativo, criado_em, atualizado_em
		FROM clients WHERE email = $1`, email)
}

// BuscarPorID retorna (cliente, nil) quando encontra, (nil, nil) quando não
// existe cliente com o id, e (nil, err) em falha real de infraestrutura.
func (r *ClientPostgres) BuscarPorID(id string) (*client.Client, error) {
	return r.buscar(`SELECT id, nome, email, telefone, senha_hash, ativo, criado_em, atualizado_em
		FROM clients WHERE id = $1`, id)
}

// Listar devolve todos os clientes com conta, ordenados por nome, para o
// painel de moderação do admin. Convidados sem conta ficam de fora.
func (r *ClientPostgres) Listar() ([]*client.Client, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, nome, email, telefone, senha_hash, ativo, criado_em, atualizado_em
		 FROM clients WHERE senha_hash IS NOT NULL ORDER BY nome`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []*client.Client
	for rows.Next() {
		c, err := escanearClient(rows)
		if err != nil {
			return nil, err
		}
		todos = append(todos, c)
	}
	return todos, rows.Err()
}

func (r *ClientPostgres) buscar(sql, arg string) (*client.Client, error) {
	c, err := escanearClient(r.pool.QueryRow(context.Background(), sql, arg))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func escanearClient(linha escaneavel) (*client.Client, error) {
	var c client.Client
	var email, telefone, senhaHash *string
	if err := linha.Scan(&c.ID, &c.Nome, &email, &telefone, &senhaHash, &c.Ativo, &c.CriadoEm, &c.AtualizadoEm); err != nil {
		return nil, err
	}
	if email != nil {
		c.Email = *email
	}
	if telefone != nil {
		c.Telefone = *telefone
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

// telefoneOuNulo converte a string vazia (cliente com conta sem telefone) para
// NULL, já que a coluna telefone é opcional no banco.
func telefoneOuNulo(telefone string) *string {
	if telefone == "" {
		return nil
	}
	return &telefone
}

// emailOuNulo converte a string vazia (convidado registrado pelo prestador,
// sem email) para NULL — assim o UNIQUE da coluna não colide entre clientes
// sem email.
func emailOuNulo(email string) *string {
	if email == "" {
		return nil
	}
	return &email
}
