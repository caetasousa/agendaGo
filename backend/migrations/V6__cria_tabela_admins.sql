-- Administradores do sistema (moderação). Não há cadastro pela API: o admin é
-- semeado no boot a partir de variáveis de ambiente. A tabela existe para
-- persistir a identidade e permitir a sessão autenticada.
CREATE TABLE admins (
    id            UUID         PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    senha_hash    VARCHAR(255) NOT NULL,
    criado_em     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
