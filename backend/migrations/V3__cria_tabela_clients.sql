CREATE TABLE clients (
    id            UUID         PRIMARY KEY,
    nome          VARCHAR(100) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    senha_hash    VARCHAR(255),
    criado_em     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
