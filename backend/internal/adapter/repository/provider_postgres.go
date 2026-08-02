package repository

import (
	"context"
	"errors"
	"strconv"

	"agendago/internal/domain/availability"
	"agendago/internal/domain/provider"
	"agendago/internal/pkg/paging"

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

	slug, err := r.slugDisponivel(ctx, tx, p.Slug)
	if err != nil {
		return err
	}
	p.Slug = slug

	_, err = tx.Exec(ctx,
		`INSERT INTO providers (id, nome, slug, aceita_agendamentos, descanso_minutos, duracao_atendimento_minutos, permite_marcacao_pelo_prestador)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.ID, p.Nome, p.Slug, p.AceitaAgendamentos, p.DescansoMinutos, p.DuracaoAtendimentoMinutos, p.PermiteMarcacaoPeloPrestador,
	)
	if err != nil {
		return err
	}

	if err := salvarHorariosPadrao(ctx, tx, p.ID, p.HorariosPadrao); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// maxTentativasSlug limita o desempate de homônimos. Passar disso significaria
// dezenas de prestadores com o mesmo nome — mais provável ser defeito do que
// caso real, e vale falhar em vez de girar.
const maxTentativasSlug = 50

// slugDisponivel desempata homônimos: "joao-silva" já tomado vira
// "joao-silva-2", depois "joao-silva-3".
//
// O desempate é do repositório, não do domínio: só quem enxerga a tabela sabe
// o que já existe. E acontece DENTRO da transação do INSERT, sob o mesmo
// contexto — duas contas com o mesmo nome criadas ao mesmo tempo ainda podem
// escolher o mesmo sufixo, e nesse caso é o UNIQUE que decide, com a segunda
// falhando de forma limpa em vez de gravar link ambíguo.
func (r *ProviderPostgres) slugDisponivel(ctx context.Context, tx pgx.Tx, base string) (string, error) {
	if base == "" {
		base = "prestador"
	}
	candidato := base
	for tentativa := 2; tentativa <= maxTentativasSlug; tentativa++ {
		var existe bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM providers WHERE slug = $1)`, candidato,
		).Scan(&existe); err != nil {
			return "", err
		}
		if !existe {
			return candidato, nil
		}
		candidato = base + "-" + strconv.Itoa(tentativa)
	}
	return "", errors.New("não foi possível gerar um endereço público único")
}

// Atualizar persiste as preferências mutáveis da agenda (ofertar horários,
// descanso, duração e expediente padrão). Não altera o nome. Telefone e senha
// não passam por aqui: são da conta, em UsuarioPostgres.
func (r *ProviderPostgres) Atualizar(p *provider.Provider) error {
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE providers
		 SET slug = $2, aceita_agendamentos = $3, descanso_minutos = $4, duracao_atendimento_minutos = $5, permite_marcacao_pelo_prestador = $6, atualizado_em = $7
		 WHERE id = $1`,
		p.ID, p.Slug, p.AceitaAgendamentos, p.DescansoMinutos, p.DuracaoAtendimentoMinutos, p.PermiteMarcacaoPeloPrestador, p.AtualizadoEm,
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
	return r.buscarPor("id", id)
}

// BuscarPorSlug retorna (prestador, nil) quando encontra, (nil, nil) quando não
// existe prestador com o slug, e (nil, err) em falha real de infraestrutura.
func (r *ProviderPostgres) BuscarPorSlug(slug string) (*provider.Provider, error) {
	return r.buscarPor("slug", slug)
}

// buscarPor monta a consulta por uma coluna única. `coluna` NUNCA vem de
// entrada do usuário — é literal nos dois chamadores — e o valor vai
// parametrizado, então não há injeção possível aqui.
func (r *ProviderPostgres) buscarPor(coluna, valor string) (*provider.Provider, error) {
	ctx := context.Background()
	var p provider.Provider
	err := r.pool.QueryRow(ctx,
		`SELECT id, nome, slug, aceita_agendamentos, descanso_minutos, duracao_atendimento_minutos, permite_marcacao_pelo_prestador, criado_em, atualizado_em
		 FROM providers WHERE `+coluna+` = $1`, valor,
	).Scan(
		&p.ID, &p.Nome, &p.Slug, &p.AceitaAgendamentos,
		&p.DescansoMinutos, &p.DuracaoAtendimentoMinutos, &p.PermiteMarcacaoPeloPrestador, &p.CriadoEm, &p.AtualizadoEm,
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

// Listar devolve uma página de prestadores (inclusive banidos, como a visão de
// moderação), ordenados por nome, e o total existente. HorariosPadrao não é
// carregado — a listagem só precisa de identificação.
func (r *ProviderPostgres) Listar(pag paging.Pagina) ([]*provider.Provider, int, error) {
	return r.listarPaginado("", pag)
}

// ListarAtivos devolve uma página de agendas cujo DONO está ativo, ordenadas
// por nome, e o total delas — a vitrine pública. Quem foi banido pelo admin
// some da vitrine, e desde a separação entre conta e agenda o banimento está
// em usuarios: por isso o join, e não um `WHERE ativo` na própria providers.
//
// O filtro roda no SQL, não em memória: com LIMIT, filtrar depois de buscar
// devolveria páginas mais curtas que o pedido e esconderia prestadores válidos
// das páginas seguintes.
func (r *ProviderPostgres) ListarAtivos(pag paging.Pagina) ([]*provider.Provider, int, error) {
	return r.listarPaginado(`
		JOIN provider_membros m ON m.provider_id = p.id AND m.papel = 'dono'
		JOIN usuarios u ON u.id = m.usuario_id
		WHERE u.ativo`, pag)
}

func (r *ProviderPostgres) listarPaginado(filtro string, pag paging.Pagina) ([]*provider.Provider, int, error) {
	pag = pag.Valida()
	ctx := context.Background()

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM providers p `+filtro).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT p.id, p.nome, p.slug, p.aceita_agendamentos, p.descanso_minutos, p.duracao_atendimento_minutos, p.permite_marcacao_pelo_prestador, p.criado_em, p.atualizado_em
		 FROM providers p `+filtro+` ORDER BY p.nome, p.id LIMIT $1 OFFSET $2`, pag.Limite, pag.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var todos []*provider.Provider
	for rows.Next() {
		var p provider.Provider
		if err := rows.Scan(
			&p.ID, &p.Nome, &p.Slug, &p.AceitaAgendamentos,
			&p.DescansoMinutos, &p.DuracaoAtendimentoMinutos, &p.PermiteMarcacaoPeloPrestador, &p.CriadoEm, &p.AtualizadoEm,
		); err != nil {
			return nil, 0, err
		}
		todos = append(todos, &p)
	}
	return todos, total, rows.Err()
}
