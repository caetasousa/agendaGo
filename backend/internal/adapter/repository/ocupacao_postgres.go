package repository

import (
	"context"
	"errors"
	"time"

	"agendago/internal/domain/ocupacao"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const colunasOcupacao = `id, provider_id, data, inicio_minutos, fim_minutos, titulo, origem, origem_externa_id, criado_em, atualizado_em`

type OcupacaoPostgres struct {
	pool *pgxpool.Pool
}

func NovoOcupacaoPostgres(pool *pgxpool.Pool) *OcupacaoPostgres {
	return &OcupacaoPostgres{pool: pool}
}

// Salvar persiste um compromisso pessoal. criado_em e atualizado_em ficam a
// cargo do DEFAULT NOW() da tabela.
func (r *OcupacaoPostgres) Salvar(o *ocupacao.Ocupacao) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO ocupacoes (id, provider_id, data, inicio_minutos, fim_minutos, titulo, origem, origem_externa_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		o.ID, o.ProviderID, o.Data, o.InicioMinutos, o.FimMinutos,
		textoOuNulo(o.Titulo), string(o.Origem), textoOuNulo(o.OrigemExternaID),
	)
	return err
}

// ListarPorPeriodo devolve os compromissos do prestador entre duas datas
// (inclusive), ordenados por data e horário. É a consulta que a oferta de
// horários usa para descontar os intervalos ocupados.
func (r *OcupacaoPostgres) ListarPorPeriodo(providerID string, de, ate time.Time) ([]*ocupacao.Ocupacao, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT `+colunasOcupacao+` FROM ocupacoes
		 WHERE provider_id = $1 AND data BETWEEN $2 AND $3
		 ORDER BY data, inicio_minutos`, providerID, de, ate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ocupacoes []*ocupacao.Ocupacao
	for rows.Next() {
		o, err := lerOcupacao(rows)
		if err != nil {
			return nil, err
		}
		ocupacoes = append(ocupacoes, o)
	}
	return ocupacoes, rows.Err()
}

// BuscarPorID retorna (ocupação, nil) quando encontra, (nil, nil) quando não
// existe, e (nil, err) em falha real de infraestrutura.
func (r *OcupacaoPostgres) BuscarPorID(id string) (*ocupacao.Ocupacao, error) {
	linha := r.pool.QueryRow(context.Background(),
		`SELECT `+colunasOcupacao+` FROM ocupacoes WHERE id = $1`, id)
	o, err := lerOcupacao(linha)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return o, nil
}

// Remover apaga um compromisso. Não erra quando o id não existe, como o DELETE
// do Postgres.
func (r *OcupacaoPostgres) Remover(id string) error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM ocupacoes WHERE id = $1`, id)
	return err
}

// leitorLinha abstrai QueryRow e Rows, que expõem o mesmo Scan.
type leitorLinha interface {
	Scan(dest ...any) error
}

func lerOcupacao(l leitorLinha) (*ocupacao.Ocupacao, error) {
	var o ocupacao.Ocupacao
	var titulo, origemExterna *string
	var origem string
	err := l.Scan(&o.ID, &o.ProviderID, &o.Data, &o.InicioMinutos, &o.FimMinutos,
		&titulo, &origem, &origemExterna, &o.CriadoEm, &o.AtualizadoEm)
	if err != nil {
		return nil, err
	}
	if titulo != nil {
		o.Titulo = *titulo
	}
	if origemExterna != nil {
		o.OrigemExternaID = *origemExterna
	}
	o.Origem = ocupacao.Origem(origem)
	return &o, nil
}

// textoOuNulo grava NULL em vez de string vazia, para a coluna opcional
// distinguir "não informado" de "informado como vazio".
func textoOuNulo(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
