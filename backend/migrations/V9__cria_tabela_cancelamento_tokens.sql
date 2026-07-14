-- Tokens de cancelamento entregues ao convidado no email de solicitação e de
-- confirmação. Guarda só o hash do token. Uso único: consumido (apagado)
-- quando o cancelamento de fato acontece. Expira num prazo generoso a partir
-- da criação — cobre o horizonte de agendamento futuro do sistema.
CREATE TABLE cancelamento_tokens (
    token_hash     CHAR(64)    PRIMARY KEY,
    appointment_id UUID        NOT NULL,
    criado_em      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expira_em      TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_cancelamento_tokens_appointment_id ON cancelamento_tokens (appointment_id);
CREATE INDEX idx_cancelamento_tokens_expira_em ON cancelamento_tokens (expira_em);
