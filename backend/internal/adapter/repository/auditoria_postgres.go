package repository

import (
	"context"
	"encoding/json"

	"agendago/internal/domain/auditoria"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditoriaPostgres grava e lê a trilha de auditoria.
//
// Não há Atualizar nem Remover, e a ausência é a garantia: a trilha é
// append-only por não existir caminho de escrita destrutiva na aplicação.
type AuditoriaPostgres struct {
	pool *pgxpool.Pool
}

func NovoAuditoriaPostgres(pool *pgxpool.Pool) *AuditoriaPostgres {
	return &AuditoriaPostgres{pool: pool}
}

// Registrar grava uma entrada na trilha.
func (r *AuditoriaPostgres) Registrar(reg *auditoria.Registro) error {
	var detalhe []byte
	if len(reg.Detalhe) > 0 {
		var err error
		if detalhe, err = json.Marshal(reg.Detalhe); err != nil {
			return err
		}
	}
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO auditoria (id, ator_id, ator_tipo, acao, alvo_tipo, alvo_id, detalhe)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		reg.ID, reg.AtorID, reg.AtorTipo, string(reg.Acao), reg.AlvoTipo, reg.AlvoID, detalhe,
	)
	return err
}

// ListarPorAlvo devolve a trilha de um alvo, da entrada mais recente para a
// mais antiga.
func (r *AuditoriaPostgres) ListarPorAlvo(alvoTipo, alvoID string, limite int) ([]*auditoria.Registro, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, ator_id, ator_tipo, acao, alvo_tipo, alvo_id, detalhe, criado_em
		 FROM auditoria WHERE alvo_tipo = $1 AND alvo_id = $2
		 ORDER BY criado_em DESC LIMIT $3`, alvoTipo, alvoID, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var registros []*auditoria.Registro
	for rows.Next() {
		var reg auditoria.Registro
		var acao string
		var detalhe []byte
		if err := rows.Scan(&reg.ID, &reg.AtorID, &reg.AtorTipo, &acao,
			&reg.AlvoTipo, &reg.AlvoID, &detalhe, &reg.CriadoEm); err != nil {
			return nil, err
		}
		reg.Acao = auditoria.Acao(acao)
		if len(detalhe) > 0 {
			if err := json.Unmarshal(detalhe, &reg.Detalhe); err != nil {
				return nil, err
			}
		}
		registros = append(registros, &reg)
	}
	return registros, rows.Err()
}
