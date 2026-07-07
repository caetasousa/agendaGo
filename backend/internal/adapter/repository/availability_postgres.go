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

// Buscar retorna (grade, nil) quando o prestador já configurou a grade
// semanal ao menos uma vez, (nil, nil) quando nunca configurou, e (nil, err)
// em falha real de infraestrutura.
func (r *AvailabilityPostgres) Buscar(providerID string) (*availability.WeeklySchedule, error) {
	ctx := context.Background()

	var scheduleID string
	err := r.pool.QueryRow(ctx,
		`SELECT id FROM weekly_schedules WHERE provider_id = $1`, providerID,
	).Scan(&scheduleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT dia_semana, inicio_minutos, fim_minutos
		 FROM weekly_schedule_blocks
		 WHERE weekly_schedule_id = $1
		 ORDER BY dia_semana, inicio_minutos`, scheduleID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dias := make(map[availability.DiaSemana][]availability.TimeBlock)
	for rows.Next() {
		var dia int
		var inicio, fim int
		if err := rows.Scan(&dia, &inicio, &fim); err != nil {
			return nil, err
		}
		diaSemana := availability.DiaSemana(dia)
		dias[diaSemana] = append(dias[diaSemana], availability.TimeBlock{InicioMinutos: inicio, FimMinutos: fim})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &availability.WeeklySchedule{ProviderID: providerID, Dias: dias}, nil
}

// Salvar substitui completamente a grade semanal do prestador: garante a
// linha âncora (upsert) e reescreve todos os blocos (delete-all + insert),
// dentro de uma transação.
func (r *AvailabilityPostgres) Salvar(s *availability.WeeklySchedule) error {
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var scheduleID string
	err = tx.QueryRow(ctx,
		`INSERT INTO weekly_schedules (id, provider_id)
		 VALUES ($1, $2)
		 ON CONFLICT (provider_id) DO UPDATE SET atualizado_em = NOW()
		 RETURNING id`,
		uuid.NewString(), s.ProviderID,
	).Scan(&scheduleID)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM weekly_schedule_blocks WHERE weekly_schedule_id = $1`, scheduleID); err != nil {
		return err
	}

	for dia, blocos := range s.Dias {
		for _, b := range blocos {
			_, err := tx.Exec(ctx,
				`INSERT INTO weekly_schedule_blocks (id, weekly_schedule_id, dia_semana, inicio_minutos, fim_minutos)
				 VALUES ($1, $2, $3, $4, $5)`,
				uuid.NewString(), scheduleID, int(dia), b.InicioMinutos, b.FimMinutos,
			)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

// BuscarPorData retorna (exceção, nil) quando encontra, (nil, nil) quando não
// existe exceção para a data, e (nil, err) em falha real de infraestrutura.
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

// BuscarPorID retorna (exceção, nil) quando encontra, (nil, nil) quando não
// existe, e (nil, err) em falha real de infraestrutura.
func (r *AvailabilityPostgres) BuscarPorID(id string) (*availability.DateException, error) {
	ctx := context.Background()
	var providerID, tipo string
	var data, criadoEm time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT provider_id, data, tipo, criado_em FROM date_exceptions WHERE id = $1`, id,
	).Scan(&providerID, &data, &tipo, &criadoEm)
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

// Listar retorna todas as exceções de data do prestador, ordenadas por data.
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

// Salvar persiste uma nova exceção de data (e seus blocos, se for do tipo extra).
func (r *AvailabilityPostgres) SalvarExcecao(e *availability.DateException) error {
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO date_exceptions (id, provider_id, data, tipo) VALUES ($1, $2, $3, $4)`,
		e.ID, e.ProviderID, e.Data, string(e.Tipo),
	)
	if err != nil {
		return err
	}

	for _, b := range e.Blocos {
		_, err := tx.Exec(ctx,
			`INSERT INTO date_exception_blocks (id, date_exception_id, inicio_minutos, fim_minutos)
			 VALUES ($1, $2, $3, $4)`,
			uuid.NewString(), e.ID, b.InicioMinutos, b.FimMinutos,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// Remover apaga a exceção de data informada. Não é erro remover uma exceção inexistente.
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
