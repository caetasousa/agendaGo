CREATE TABLE sessions (
    token_hash CHAR(64)    PRIMARY KEY,
    user_id    UUID        NOT NULL,
    user_type  VARCHAR(10) NOT NULL,
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expira_em  TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_sessions_expira_em ON sessions (expira_em);
