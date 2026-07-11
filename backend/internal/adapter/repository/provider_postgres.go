package repository

import (
	"context"
	"errors"

	"agendago/internal/domain/availability"
	"agendago/internal/domain/provider"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProviderPostgres struct {
	pool *pgxpool.Pool
}

func NovoProviderPostgres(pool *pgxpool.Pool) *ProviderPostgres {
	return &ProviderPostgres{pool: pool}
}

// Salvar persiste um novo prestador e os blocos do seu expediente padrão.
// criado_em e atualizado_em ficam a cargo do DEFAULT NOW() da tabela — por
// isso não são enviados no INSERT.
func (r *ProviderPostgres) Salvar(p *provider.Provider) error {
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO providers (id, nome, email, senha_hash, aceita_agendamentos, descanso_minutos)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.Nome, p.Email, p.SenhaHash, p.AceitaAgendamentos, p.DescansoMinutos,
	)
	if err != nil {
		return err
	}

	if err := salvarHorariosPadrao(ctx, tx, p.ID, p.HorariosPadrao); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// BuscarPorEmail retorna (prestador, nil) quando encontra, (nil, nil) quando
// não existe prestador com o email, e (nil, err) em falha real de infraestrutura.
func (r *ProviderPostgres) BuscarPorEmail(email string) (*provider.Provider, error) {
	ctx := context.Background()
	var p provider.Provider
	err := r.pool.QueryRow(ctx,
		`SELECT id, nome, email, senha_hash, aceita_agendamentos, descanso_minutos, criado_em, atualizado_em
		 FROM providers WHERE email = $1`, email,
	).Scan(
		&p.ID, &p.Nome, &p.Email, &p.SenhaHash, &p.AceitaAgendamentos,
		&p.DescansoMinutos, &p.CriadoEm, &p.AtualizadoEm,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if p.HorariosPadrao, err = r.buscarHorariosPadrao(ctx, p.ID); err != nil {
		return nil, err
	}
	return &p, nil
}

// Atualizar persiste as preferências mutáveis do prestador (agenda, descanso
// e expediente padrão). Não altera nome, email ou senha.
func (r *ProviderPostgres) Atualizar(p *provider.Provider) error {
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE providers
		 SET aceita_agendamentos = $2, descanso_minutos = $3, atualizado_em = $4
		 WHERE id = $1`,
		p.ID, p.AceitaAgendamentos, p.DescansoMinutos, p.AtualizadoEm,
	)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM horarios_padrao WHERE provider_id = $1`, p.ID); err != nil {
		return err
	}
	if err := salvarHorariosPadrao(ctx, tx, p.ID, p.HorariosPadrao); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// BuscarPorID retorna (prestador, nil) quando encontra, (nil, nil) quando não
// existe prestador com o id, e (nil, err) em falha real de infraestrutura.
func (r *ProviderPostgres) BuscarPorID(id string) (*provider.Provider, error) {
	ctx := context.Background()
	var p provider.Provider
	err := r.pool.QueryRow(ctx,
		`SELECT id, nome, email, senha_hash, aceita_agendamentos, descanso_minutos, criado_em, atualizado_em
		 FROM providers WHERE id = $1`, id,
	).Scan(
		&p.ID, &p.Nome, &p.Email, &p.SenhaHash, &p.AceitaAgendamentos,
		&p.DescansoMinutos, &p.CriadoEm, &p.AtualizadoEm,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if p.HorariosPadrao, err = r.buscarHorariosPadrao(ctx, p.ID); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProviderPostgres) buscarHorariosPadrao(ctx context.Context, providerID string) ([]availability.TimeBlock, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT inicio_minutos, fim_minutos FROM horarios_padrao WHERE provider_id = $1 ORDER BY inicio_minutos`,
		providerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocos []availability.TimeBlock
	for rows.Next() {
		var inicio, fim int
		if err := rows.Scan(&inicio, &fim); err != nil {
			return nil, err
		}
		blocos = append(blocos, availability.TimeBlock{InicioMinutos: inicio, FimMinutos: fim})
	}
	return blocos, rows.Err()
}

func salvarHorariosPadrao(ctx context.Context, tx pgx.Tx, providerID string, blocos []availability.TimeBlock) error {
	for _, b := range blocos {
		_, err := tx.Exec(ctx,
			`INSERT INTO horarios_padrao (id, provider_id, inicio_minutos, fim_minutos)
			 VALUES ($1, $2, $3, $4)`,
			uuid.NewString(), providerID, b.InicioMinutos, b.FimMinutos,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
