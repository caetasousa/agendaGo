-- Tokens de cancelamento entregues ao convidado no email de confirmação.
-- Guarda só o hash do token. Sem expira_em: o token vale enquanto o
-- agendamento for cancelável (a regra de antecedência vive no domínio).
CREATE TABLE cancelamento_tokens (
    token_hash     CHAR(64)    PRIMARY KEY,
    appointment_id UUID        NOT NULL,
    criado_em      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cancelamento_tokens_appointment_id ON cancelamento_tokens (appointment_id);
