package repository

import (
	"context"
	"errors"
	"time"

	"agendago/internal/domain/convite"
	"agendago/internal/domain/membro"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConvitePostgres struct {
	pool *pgxpool.Pool
}

func NovoConvitePostgres(pool *pgxpool.Pool) *ConvitePostgres {
	return &ConvitePostgres{pool: pool}
}

// Salvar persiste um convite, substituindo o que existir para o mesmo email e
// agenda. Reconvidar alguém emite um token novo e invalida o anterior — dois
// links válidos para o mesmo destino só serviriam para confundir.
func (r *ConvitePostgres) Salvar(c *convite.Convite) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO convites_membro (token_hash, email, provider_id, papel, expira_em)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (email, provider_id) DO UPDATE
		 SET token_hash = EXCLUDED.token_hash,
		     papel      = EXCLUDED.papel,
		     criado_em  = NOW(),
		     expira_em  = EXCLUDED.expira_em`,
		c.TokenHash, c.Email, c.ProviderID, string(c.Papel), c.ExpiraEm,
	)
	return err
}

// BuscarPorTokenHash lê o convite sem consumi-lo: a tela de aceite mostra de
// quem é a agenda antes de a pessoa decidir. Retorna (nil, nil) quando não
// existe.
func (r *ConvitePostgres) BuscarPorTokenHash(hash string) (*convite.Convite, error) {
	var c convite.Convite
	var papel string
	err := r.pool.QueryRow(context.Background(),
		`SELECT token_hash, email, provider_id, papel, criado_em, expira_em
		 FROM convites_membro WHERE token_hash = $1`, hash,
	).Scan(&c.TokenHash, &c.Email, &c.ProviderID, &papel, &c.CriadoEm, &c.ExpiraEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.Papel = membro.Papel(papel)
	return &c, nil
}

// Consumir apaga o convite e devolve o que estava lá, tornando o token de uso
// único. A leitura e a remoção são a mesma instrução para que dois aceites
// simultâneos não criem dois vínculos.
func (r *ConvitePostgres) Consumir(hash string) (*convite.Convite, error) {
	var c convite.Convite
	var papel string
	err := r.pool.QueryRow(context.Background(),
		`DELETE FROM convites_membro WHERE token_hash = $1
		 RETURNING token_hash, email, provider_id, papel, criado_em, expira_em`, hash,
	).Scan(&c.TokenHash, &c.Email, &c.ProviderID, &papel, &c.CriadoEm, &c.ExpiraEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.Papel = membro.Papel(papel)
	return &c, nil
}

// ListarPendentesPorProvider devolve os convites ainda não aceitos e dentro da
// validade, para a tela de equipe mostrar quem falta entrar.
func (r *ConvitePostgres) ListarPendentesPorProvider(providerID string) ([]*convite.Convite, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT token_hash, email, provider_id, papel, criado_em, expira_em
		 FROM convites_membro
		 WHERE provider_id = $1 AND expira_em > NOW()
		 ORDER BY criado_em`, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convites []*convite.Convite
	for rows.Next() {
		var c convite.Convite
		var papel string
		if err := rows.Scan(&c.TokenHash, &c.Email, &c.ProviderID, &papel, &c.CriadoEm, &c.ExpiraEm); err != nil {
			return nil, err
		}
		c.Papel = membro.Papel(papel)
		convites = append(convites, &c)
	}
	return convites, rows.Err()
}

// RemoverPorEmail cancela um convite pendente antes de ele ser aceito.
func (r *ConvitePostgres) RemoverPorEmail(providerID, email string) error {
	_, err := r.pool.Exec(context.Background(),
		`DELETE FROM convites_membro WHERE provider_id = $1 AND email = $2`, providerID, email)
	return err
}

// RemoverExpirados limpa os convites vencidos. Chamado de carona nas operações
// de convite, como os demais tokens do sistema.
func (r *ConvitePostgres) RemoverExpirados() error {
	_, err := r.pool.Exec(context.Background(),
		`DELETE FROM convites_membro WHERE expira_em < $1`, time.Now())
	return err
}
