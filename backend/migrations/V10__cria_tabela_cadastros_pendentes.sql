-- Cadastros de cliente aguardando confirmação por email. Guarda os dados da
-- conta a criar (com a senha já hasheada, nunca em texto puro) e só o hash do
-- token. Ao confirmar, a conta é criada (ou o convidado é convertido) e o
-- registro é consumido.
CREATE TABLE cadastros_pendentes (
    token_hash CHAR(64)     PRIMARY KEY,
    nome       VARCHAR(100) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    telefone   VARCHAR(30)  NOT NULL,
    senha_hash VARCHAR(255) NOT NULL,
    criado_em  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expira_em  TIMESTAMPTZ  NOT NULL
);

CREATE INDEX idx_cadastros_pendentes_email ON cadastros_pendentes (email);
CREATE INDEX idx_cadastros_pendentes_expira_em ON cadastros_pendentes (expira_em);
