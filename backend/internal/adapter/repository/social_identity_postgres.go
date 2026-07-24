package repository

import (
	"context"
	"errors"

	"agendago/internal/domain/socialidentity"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SocialIdentityPostgres struct {
	pool *pgxpool.Pool
}

func NovoSocialIdentityPostgres(pool *pgxpool.Pool) *SocialIdentityPostgres {
	return &SocialIdentityPostgres{pool: pool}
}

// Salvar persiste uma nova identidade social.
func (r *SocialIdentityPostgres) Salvar(i *socialidentity.Identidade) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO social_identities (id, provedor, sub, user_id, user_type, email, criado_em)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		i.ID, string(i.Provedor), i.Sub, i.UserID, i.UserType, emailOuNulo(i.Email), i.CriadoEm,
	)
	return err
}

// BuscarPorProvedorSub retorna (identidade, nil) quando encontra, (nil, nil)
// quando não existe identidade com o (provedor, sub), e (nil, err) em falha
// real de infraestrutura.
func (r *SocialIdentityPostgres) BuscarPorProvedorSub(provedor socialidentity.Provedor, sub string) (*socialidentity.Identidade, error) {
	var i socialidentity.Identidade
	var provedorStr string
	var email *string
	err := r.pool.QueryRow(context.Background(),
		`SELECT id, provedor, sub, user_id, user_type, email, criado_em
		 FROM social_identities WHERE provedor = $1 AND sub = $2`, string(provedor), sub,
	).Scan(&i.ID, &provedorStr, &i.Sub, &i.UserID, &i.UserType, &email, &i.CriadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	i.Provedor = socialidentity.Provedor(provedorStr)
	if email != nil {
		i.Email = *email
	}
	return &i, nil
}

// RemoverDoUsuario apaga todas as identidades sociais de um usuário — usado
// ao bani-lo ou apagar sua conta.
func (r *SocialIdentityPostgres) RemoverDoUsuario(userID string) error {
	_, err := r.pool.Exec(context.Background(),
		`DELETE FROM social_identities WHERE user_id = $1`, userID,
	)
	return err
}
