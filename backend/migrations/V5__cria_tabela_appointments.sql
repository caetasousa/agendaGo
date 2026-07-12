-- Agendamentos entre cliente e prestador: a reserva de um intervalo de uma
-- data (minutos desde a meia-noite, fuso único do sistema). A máquina de
-- estados, o TTL e o anti-overbooking (lock transacional + checagem de
-- conflito) são responsabilidade do domínio/usecase, não do banco.
CREATE TABLE appointments (
    id             UUID        PRIMARY KEY,
    provider_id    UUID        NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    client_id      UUID        NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    data           DATE        NOT NULL,
    inicio_minutos SMALLINT    NOT NULL,
    fim_minutos    SMALLINT    NOT NULL,
    status         VARCHAR(20) NOT NULL,
    expira_em      TIMESTAMPTZ NOT NULL,
    criado_em      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    atualizado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_appointments_provider_data ON appointments (provider_id, data);
CREATE INDEX idx_appointments_client ON appointments (client_id);
