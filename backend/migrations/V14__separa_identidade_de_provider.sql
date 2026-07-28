-- Separa QUEM LOGA de QUAL AGENDA. Até aqui `providers` acumulava os dois
-- papéis: identidade de login (email, senha_hash, telefone, ativo) e entidade
-- de negócio (horários, duração, buffer). Enquanto for assim, uma segunda
-- pessoa — recepcionista, sócia — só opera a agenda compartilhando a senha.
--
-- `usuarios` passa a ser a identidade; `providers` continua sendo a agenda, e
-- por isso appointments.provider_id, date_exceptions.provider_id e
-- horarios_padrao.provider_id NÃO são tocados aqui. `provider_membros` liga
-- as duas pontas e é onde mora o papel de cada pessoa numa agenda.
--
-- Esta migration REMOVE as colunas de identidade de `providers` no mesmo passo
-- em que cria as tabelas novas. Não é expand/contract: o banco de produção é
-- descartável neste momento do projeto, então não há dado a preservar entre
-- duas versões do código nem janela em que as duas precisem conviver. Quando
-- existir base real com usuários de verdade, uma mudança equivalente terá que
-- ser dividida em dois deploys — ver docs/tecnologias.md.
--
-- `papel` é VARCHAR sem CHECK de propósito: quais papéis existem e o que cada
-- um pode fazer é decisão do domínio (internal/domain/membro), não do banco.
CREATE TABLE usuarios (
    id            UUID         PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    senha_hash    VARCHAR(255) NOT NULL,
    telefone      VARCHAR(30)  NOT NULL,
    ativo         BOOLEAN      NOT NULL,
    criado_em     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE provider_membros (
    id          UUID        PRIMARY KEY,
    usuario_id  UUID        NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    provider_id UUID        NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    papel       VARCHAR(20) NOT NULL,
    criado_em   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (usuario_id, provider_id)
);

CREATE INDEX idx_provider_membros_usuario ON provider_membros (usuario_id);

-- Converte os prestadores que porventura existam antes de remover as colunas.
-- Em produção e no CI a tabela está vazia e isto não copia nada; num banco de
-- desenvolvimento com dados, evita perdê-los sem querer.
--
-- O id do usuário é o MESMO id do provider, deliberadamente. sessions.user_id
-- (V3) e social_identities.user_id (V11) apontam para o id do prestador; ao
-- reusá-lo, essas duas tabelas seguem coerentes sem serem migradas.
INSERT INTO usuarios (id, email, senha_hash, telefone, ativo, criado_em, atualizado_em)
SELECT id, email, senha_hash, telefone, ativo, criado_em, atualizado_em
FROM providers;

-- gen_random_uuid() é built-in no Postgres 13+. Aqui ele gera chave substituta
-- para linhas criadas pela própria migração — não é DEFAULT de regra de
-- negócio: o id de todo vínculo criado pela aplicação vem do domínio.
--
-- Todo prestador convertido vira dono da própria agenda: era a única relação
-- possível antes desta migration.
INSERT INTO provider_membros (id, usuario_id, provider_id, papel, criado_em)
SELECT gen_random_uuid(), id, id, 'dono', criado_em
FROM providers;

-- Identidade sai de `providers` para valer. O UNIQUE de email vai junto com a
-- coluna — a unicidade do email agora é responsabilidade de `usuarios`.
ALTER TABLE providers
    DROP COLUMN email,
    DROP COLUMN senha_hash,
    DROP COLUMN telefone,
    DROP COLUMN ativo;
