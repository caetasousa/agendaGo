-- Tokens de pré-cadastro: carregam nome/email/telefone de um convidado do
-- email direto para a tela de cadastro, para pré-preencher o formulário sem
-- redigitar e criar a conta sem uma segunda confirmação por email. Uso único
-- (o submit final consome). Expira num prazo curto a partir da criação.
CREATE TABLE pre_cadastro_tokens (
    token_hash CHAR(64)     PRIMARY KEY,
    nome       VARCHAR(100) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    telefone   VARCHAR(30)  NOT NULL,
    criado_em  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expira_em  TIMESTAMPTZ  NOT NULL
);

CREATE INDEX idx_pre_cadastro_tokens_expira_em ON pre_cadastro_tokens (expira_em);
