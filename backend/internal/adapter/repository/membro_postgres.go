package repository

import (
	"context"
	"errors"

	"agendago/internal/domain/membro"
	"agendago/internal/domain/usuario"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const colunasMembro = `id, usuario_id, provider_id, papel, criado_em`

type MembroPostgres struct {
	pool *pgxpool.Pool
}

func NovoMembroPostgres(pool *pgxpool.Pool) *MembroPostgres {
	return &MembroPostgres{pool: pool}
}

// Salvar persiste um novo vínculo entre usuário e agenda. criado_em fica a
// cargo do DEFAULT NOW() da tabela.
func (r *MembroPostgres) Salvar(m *membro.Membro) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO provider_membros (id, usuario_id, provider_id, papel)
		 VALUES ($1, $2, $3, $4)`,
		m.ID, m.UsuarioID, m.ProviderID, string(m.Papel),
	)
	return err
}

// BuscarPorUsuario devolve o vínculo ativo de um usuário — a agenda que ele
// opera. Enquanto cada pessoa tiver exatamente um vínculo a resposta é
// determinística; quando houver mais de um, é aqui que entra a escolha de com
// qual agenda se está operando. Ordena por criado_em para que a resposta não
// dependa da ordem física das linhas nesse meio-tempo.
//
// Retorna (vínculo, nil) quando encontra, (nil, nil) quando o usuário não tem
// vínculo, e (nil, err) em falha real de infraestrutura.
func (r *MembroPostgres) BuscarPorUsuario(usuarioID string) (*membro.Membro, error) {
	var m membro.Membro
	var papel string
	err := r.pool.QueryRow(context.Background(),
		`SELECT `+colunasMembro+` FROM provider_membros
		 WHERE usuario_id = $1 ORDER BY criado_em, id LIMIT 1`, usuarioID,
	).Scan(&m.ID, &m.UsuarioID, &m.ProviderID, &papel, &m.CriadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.Papel = membro.Papel(papel)
	return &m, nil
}

// DonoAtivo informa se a agenda tem dono e se esse dono não está banido. É o
// que decide se ela aparece na vitrine e se oferta horários: desde a separação
// entre conta e agenda, o banimento mora em usuarios, e a agenda sozinha não
// sabe responder isso.
//
// Uma agenda sem dono responde false — não deveria existir, mas se existir o
// mais seguro é não ofertar nada.
func (r *MembroPostgres) DonoAtivo(providerID string) (bool, error) {
	var ativo bool
	err := r.pool.QueryRow(context.Background(),
		`SELECT EXISTS (
			SELECT 1 FROM provider_membros m
			JOIN usuarios u ON u.id = m.usuario_id
			WHERE m.provider_id = $1 AND m.papel = 'dono' AND u.ativo
		)`, providerID).Scan(&ativo)
	return ativo, err
}

// DonoDe devolve a conta dona de uma agenda, ou (nil, nil) quando a agenda não
// tem dono. Usado onde é preciso mais de um campo da conta — a moderação, por
// exemplo, que mostra email e situação lado a lado com os dados da agenda.
func (r *MembroPostgres) DonoDe(providerID string) (*usuario.Usuario, error) {
	var u usuario.Usuario
	err := r.pool.QueryRow(context.Background(),
		`SELECT `+colunasUsuarioComPrefixo+` FROM provider_membros m
		 JOIN usuarios u ON u.id = m.usuario_id
		 WHERE m.provider_id = $1 AND m.papel = 'dono'`, providerID,
	).Scan(&u.ID, &u.Email, &u.SenhaHash, &u.Telefone, &u.Ativo, &u.CriadoEm, &u.AtualizadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// EmailDoDono devolve o email para onde vão as notificações de uma agenda.
// Hoje é o do dono; quando existir mais de um membro, é aqui que se decide
// quem recebe. Devolve string vazia quando a agenda não tem dono, e cabe a
// quem chama tratar isso como "não há para quem notificar" — perder um email
// é melhor do que derrubar o agendamento por causa dele.
func (r *MembroPostgres) EmailDoDono(providerID string) (string, error) {
	var email string
	err := r.pool.QueryRow(context.Background(),
		`SELECT u.email FROM provider_membros m
		 JOIN usuarios u ON u.id = m.usuario_id
		 WHERE m.provider_id = $1 AND m.papel = 'dono'`, providerID).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return email, err
}

// ListarPorProvider devolve todos os vínculos de uma agenda, ordenados por
// criação — quem tem acesso a ela e com qual papel.
func (r *MembroPostgres) ListarPorProvider(providerID string) ([]*membro.Membro, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT `+colunasMembro+` FROM provider_membros
		 WHERE provider_id = $1 ORDER BY criado_em, id`, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var membros []*membro.Membro
	for rows.Next() {
		var m membro.Membro
		var papel string
		if err := rows.Scan(&m.ID, &m.UsuarioID, &m.ProviderID, &papel, &m.CriadoEm); err != nil {
			return nil, err
		}
		m.Papel = membro.Papel(papel)
		membros = append(membros, &m)
	}
	return membros, rows.Err()
}
