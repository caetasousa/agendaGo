package repository

import (
	"context"
	"errors"
	"time"

	"agendago/internal/domain/availability"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AvailabilityPostgres struct {
	pool *pgxpool.Pool
}

func NovoAvailabilityPostgres(pool *pgxpool.Pool) *AvailabilityPostgres {
	return &AvailabilityPostgres{pool: pool}
}

// BuscarPorData retorna (definição, nil) quando encontra, (nil, nil) quando a
// data não tem definição própria, e (nil, err) em falha real de infraestrutura.
func (r *AvailabilityPostgres) BuscarPorData(providerID string, data time.Time) (*availability.DateException, error) {
	ctx := context.Background()
	var id, tipo string
	var criadoEm time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT id, tipo, criado_em FROM date_exceptions WHERE provider_id = $1 AND data = $2`,
		providerID, data,
	).Scan(&id, &tipo, &criadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	blocos, err := r.buscarBlocosDaExcecao(ctx, id)
	if err != nil {
		return nil, err
	}

	return &availability.DateException{
		ID: id, ProviderID: providerID, Data: data,
		Tipo: availability.TipoExcecao(tipo), Blocos: blocos, CriadoEm: criadoEm,
	}, nil
}

// Listar retorna todas as definições de data do prestador, ordenadas por data.
func (r *AvailabilityPostgres) Listar(providerID string) ([]*availability.DateException, error) {
	ctx := context.Background()
	rows, err := r.pool.Query(ctx,
		`SELECT id, data, tipo, criado_em FROM date_exceptions WHERE provider_id = $1 ORDER BY data`,
		providerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var excecoes []*availability.DateException
	for rows.Next() {
		var id, tipo string
		var data, criadoEm time.Time
		if err := rows.Scan(&id, &data, &tipo, &criadoEm); err != nil {
			return nil, err
		}
		blocos, err := r.buscarBlocosDaExcecao(ctx, id)
		if err != nil {
			return nil, err
		}
		excecoes = append(excecoes, &availability.DateException{
			ID: id, ProviderID: providerID, Data: data,
			Tipo: availability.TipoExcecao(tipo), Blocos: blocos, CriadoEm: criadoEm,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return excecoes, nil
}

// SalvarExcecao persiste a definição de uma data (upsert por provider+data):
// em conflito, atualiza o tipo e reescreve os blocos dentro de uma transação.
func (r *AvailabilityPostgres) SalvarExcecao(e *availability.DateException) error {
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Em conflito o id original da linha é mantido; blocos são reescritos
	// usando o id retornado, não o do parâmetro.
	var id string
	err = tx.QueryRow(ctx,
		`INSERT INTO date_exceptions (id, provider_id, data, tipo)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (provider_id, data) DO UPDATE SET tipo = EXCLUDED.tipo
		 RETURNING id`,
		e.ID, e.ProviderID, e.Data, string(e.Tipo),
	).Scan(&id)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM date_exception_blocks WHERE date_exception_id = $1`, id); err != nil {
		return err
	}

	for _, b := range e.Blocos {
		_, err := tx.Exec(ctx,
			`INSERT INTO date_exception_blocks (id, date_exception_id, inicio_minutos, fim_minutos)
			 VALUES ($1, $2, $3, $4)`,
			uuid.NewString(), id, b.InicioMinutos, b.FimMinutos,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// Remover apaga a definição de data informada. Não é erro remover uma definição inexistente.
func (r *AvailabilityPostgres) Remover(id string) error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM date_exceptions WHERE id = $1`, id)
	return err
}

func (r *AvailabilityPostgres) buscarBlocosDaExcecao(ctx context.Context, excecaoID string) ([]availability.TimeBlock, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT inicio_minutos, fim_minutos FROM date_exception_blocks WHERE date_exception_id = $1 ORDER BY inicio_minutos`,
		excecaoID,
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
