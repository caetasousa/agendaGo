-- Prestadores de serviço e os blocos do seu expediente padrão (configurável
-- em Preferências). Validações de negócio — faixas de minutos, fim > início,
-- granularidade — são responsabilidade do domínio, não do banco.
CREATE TABLE providers (
    id                   UUID         PRIMARY KEY,
    nome                 VARCHAR(100) NOT NULL,
    email                VARCHAR(255) NOT NULL UNIQUE,
    senha_hash           VARCHAR(255) NOT NULL,
    aceita_agendamentos  BOOLEAN      NOT NULL,
    descanso_minutos     INT          NOT NULL,
    criado_em            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    atualizado_em        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE horarios_padrao (
    id             UUID     PRIMARY KEY,
    provider_id    UUID     NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    inicio_minutos SMALLINT NOT NULL,
    fim_minutos    SMALLINT NOT NULL
);

CREATE INDEX idx_horarios_padrao_provider ON horarios_padrao (provider_id);
