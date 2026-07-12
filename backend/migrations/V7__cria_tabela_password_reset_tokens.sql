-- Tokens de recuperação de senha, de uso único. Guarda só o hash do token,
-- como a tabela sessions.
CREATE TABLE password_reset_tokens (
    token_hash CHAR(64)    PRIMARY KEY,
    user_id    UUID        NOT NULL,
    user_type  VARCHAR(10) NOT NULL,
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expira_em  TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens (user_id);
CREATE INDEX idx_password_reset_tokens_expira_em ON password_reset_tokens (expira_em);
