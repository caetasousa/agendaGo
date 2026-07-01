CREATE TABLE providers (
    id                   UUID        PRIMARY KEY,
    nome                 VARCHAR(100) NOT NULL,
    email                VARCHAR(255) NOT NULL UNIQUE,
    senha                VARCHAR(255) NOT NULL,
    aceita_agendamentos  BOOLEAN     NOT NULL,
    descanso_minutos     INT         NOT NULL,
    criado_em            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    atualizado_em        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
