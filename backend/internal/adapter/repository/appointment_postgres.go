package repository

import (
	"context"
	"errors"
	"time"

	"agendago/internal/domain/appointment"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AppointmentPostgres struct {
	pool *pgxpool.Pool
}

func NovoAppointmentPostgres(pool *pgxpool.Pool) *AppointmentPostgres {
	return &AppointmentPostgres{pool: pool}
}

// SalvarSeLivre persiste a solicitação somente se o intervalo não colidir com
// outro agendamento que ocupa horário. O anti-overbooking é garantido por
// transação: um lock na linha do prestador serializa as reservas concorrentes
// dele, e a checagem de conflito + INSERT acontecem sob esse lock — sem
// regra de negócio no schema.
func (r *AppointmentPostgres) SalvarSeLivre(a *appointment.Appointment, agora time.Time) error {
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT 1 FROM providers WHERE id = $1 FOR UPDATE`, a.ProviderID); err != nil {
		return err
	}

	var conflito bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM appointments
			WHERE provider_id = $1 AND data = $2
			  AND inicio_minutos < $4 AND $3 < fim_minutos
			  AND (status = $5 OR (status = $6 AND expira_em > $7))
		)`,
		a.ProviderID, a.Data, a.InicioMinutos, a.FimMinutos,
		string(appointment.StatusConfirmado), string(appointment.StatusSolicitado), agora,
	).Scan(&conflito)
	if err != nil {
		return err
	}
	if conflito {
		return appointment.ErrConflitoHorario
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO appointments (id, provider_id, client_id, data, inicio_minutos, fim_minutos, status, expira_em, criado_em, atualizado_em)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		a.ID, a.ProviderID, a.ClientID, a.Data, a.InicioMinutos, a.FimMinutos,
		string(a.Status), a.ExpiraEm, a.CriadoEm, a.AtualizadoEm,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// BuscarPorID retorna (agendamento, nil) quando encontra, (nil, nil) quando
// não existe, e (nil, err) em falha real de infraestrutura.
func (r *AppointmentPostgres) BuscarPorID(id string) (*appointment.Appointment, error) {
	linha := r.pool.QueryRow(context.Background(),
		`SELECT id, provider_id, client_id, data, inicio_minutos, fim_minutos, status, expira_em, criado_em, atualizado_em
		 FROM appointments WHERE id = $1`, id)

	a, err := escanearAppointment(linha)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// Atualizar persiste o estado atual do agendamento (status e atualizado_em).
func (r *AppointmentPostgres) Atualizar(a *appointment.Appointment) error {
	_, err := r.pool.Exec(context.Background(),
		`UPDATE appointments SET status = $2, atualizado_em = $3 WHERE id = $1`,
		a.ID, string(a.Status), a.AtualizadoEm,
	)
	return err
}

// ListarPorPrestador devolve os agendamentos do prestador ordenados por data e início.
func (r *AppointmentPostgres) ListarPorPrestador(providerID string) ([]*appointment.Appointment, error) {
	return r.listar(
		`SELECT id, provider_id, client_id, data, inicio_minutos, fim_minutos, status, expira_em, criado_em, atualizado_em
		 FROM appointments WHERE provider_id = $1 ORDER BY data, inicio_minutos`, providerID)
}

// ListarPorCliente devolve os agendamentos do cliente ordenados por data e início.
func (r *AppointmentPostgres) ListarPorCliente(clientID string) ([]*appointment.Appointment, error) {
	return r.listar(
		`SELECT id, provider_id, client_id, data, inicio_minutos, fim_minutos, status, expira_em, criado_em, atualizado_em
		 FROM appointments WHERE client_id = $1 ORDER BY data, inicio_minutos`, clientID)
}

// ListarOcupantesPorPeriodo devolve os agendamentos do prestador que ocupam
// horário (SOLICITADO não expirado ou CONFIRMADO) entre as datas, inclusive.
func (r *AppointmentPostgres) ListarOcupantesPorPeriodo(providerID string, de, ate time.Time, agora time.Time) ([]*appointment.Appointment, error) {
	return r.listar(
		`SELECT id, provider_id, client_id, data, inicio_minutos, fim_minutos, status, expira_em, criado_em, atualizado_em
		 FROM appointments
		 WHERE provider_id = $1 AND data BETWEEN $2 AND $3
		   AND (status = $4 OR (status = $5 AND expira_em > $6))
		 ORDER BY data, inicio_minutos`,
		providerID, de, ate,
		string(appointment.StatusConfirmado), string(appointment.StatusSolicitado), agora)
}

// ListarConfirmadosSemLembrete devolve os agendamentos CONFIRMADOs cuja data
// está entre de e ate (inclusive) e cujo lembrete ainda não foi enviado.
func (r *AppointmentPostgres) ListarConfirmadosSemLembrete(de, ate time.Time) ([]*appointment.Appointment, error) {
	return r.listar(
		`SELECT id, provider_id, client_id, data, inicio_minutos, fim_minutos, status, expira_em, criado_em, atualizado_em
		 FROM appointments
		 WHERE status = $1 AND data BETWEEN $2 AND $3 AND lembrete_enviado_em IS NULL
		 ORDER BY data, inicio_minutos`,
		string(appointment.StatusConfirmado), de, ate)
}

// MarcarLembreteEnviado marca o lembrete como enviado, mas só se ainda não
// tiver sido — o UPDATE condicional funciona como claim: quando duas
// execuções competem pelo mesmo agendamento, só uma tem RowsAffected() > 0,
// o que evita lembrete duplicado sem exigir lock explícito.
func (r *AppointmentPostgres) MarcarLembreteEnviado(id string, quando time.Time) (bool, error) {
	tag, err := r.pool.Exec(context.Background(),
		`UPDATE appointments SET lembrete_enviado_em = $2 WHERE id = $1 AND lembrete_enviado_em IS NULL`,
		id, quando,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *AppointmentPostgres) listar(sql string, args ...any) ([]*appointment.Appointment, error) {
	rows, err := r.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resultado []*appointment.Appointment
	for rows.Next() {
		a, err := escanearAppointment(rows)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, a)
	}
	return resultado, rows.Err()
}

type escaneavel interface {
	Scan(dest ...any) error
}

func escanearAppointment(linha escaneavel) (*appointment.Appointment, error) {
	var a appointment.Appointment
	var status string
	err := linha.Scan(
		&a.ID, &a.ProviderID, &a.ClientID, &a.Data, &a.InicioMinutos, &a.FimMinutos,
		&status, &a.ExpiraEm, &a.CriadoEm, &a.AtualizadoEm,
	)
	if err != nil {
		return nil, err
	}
	a.Status = appointment.Status(status)
	return &a, nil
}
