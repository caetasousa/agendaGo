-- Clientes. senha_hash é opcional: um cliente pode existir sem conta
-- (cadastrado pelo prestador) e criar a conta depois.
CREATE TABLE clients (
    id            UUID         PRIMARY KEY,
    nome          VARCHAR(100) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    senha_hash    VARCHAR(255),
    criado_em     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
