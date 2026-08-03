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
-- em que cria as tabelas novas, e não converte prestador nenhum: o banco é
-- descartável neste momento do projeto, então não há dado a preservar entre
-- duas versões do código nem janela em que as duas precisem conviver. Converter
-- exigiria ainda decidir em SQL que todo prestador vira dono da própria agenda
-- — regra do domínio, escrita onde não há teste que a cubra. Quando existir
-- base real com usuários de verdade, uma mudança equivalente terá que ser
-- dividida em dois deploys — ver docs/tecnologias.md.
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

-- Identidade sai de `providers` para valer. O UNIQUE de email vai junto com a
-- coluna — a unicidade do email agora é responsabilidade de `usuarios`.
ALTER TABLE providers
    DROP COLUMN email,
    DROP COLUMN senha_hash,
    DROP COLUMN telefone,
    DROP COLUMN ativo;
