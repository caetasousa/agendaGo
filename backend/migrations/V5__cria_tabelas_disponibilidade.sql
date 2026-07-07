-- Linha âncora: existe assim que o prestador salva a grade ao menos uma vez
-- (mesmo vazia em todos os dias). Distingue "grade configurada e vazia" de
-- "grade nunca configurada" — usado na resolução de disponibilidade para
-- decidir se cai no default comercial.
CREATE TABLE weekly_schedules (
    id            UUID        PRIMARY KEY,
    provider_id   UUID        NOT NULL UNIQUE REFERENCES providers(id) ON DELETE CASCADE,
    criado_em     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE weekly_schedule_blocks (
    id                 UUID     PRIMARY KEY,
    weekly_schedule_id UUID     NOT NULL REFERENCES weekly_schedules(id) ON DELETE CASCADE,
    dia_semana         SMALLINT NOT NULL CHECK (dia_semana BETWEEN 0 AND 6),
    inicio_minutos     SMALLINT NOT NULL CHECK (inicio_minutos >= 0 AND inicio_minutos < 1440),
    fim_minutos        SMALLINT NOT NULL CHECK (fim_minutos > 0 AND fim_minutos <= 1440),
    CHECK (fim_minutos > inicio_minutos)
);

CREATE INDEX idx_weekly_schedule_blocks_schedule_dia
    ON weekly_schedule_blocks (weekly_schedule_id, dia_semana);

CREATE TABLE date_exceptions (
    id          UUID        PRIMARY KEY,
    provider_id UUID        NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    data        DATE        NOT NULL,
    tipo        VARCHAR(10) NOT NULL CHECK (tipo IN ('bloqueio', 'extra')),
    criado_em   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider_id, data)
);

CREATE INDEX idx_date_exceptions_provider_data ON date_exceptions (provider_id, data);

CREATE TABLE date_exception_blocks (
    id                 UUID     PRIMARY KEY,
    date_exception_id  UUID     NOT NULL REFERENCES date_exceptions(id) ON DELETE CASCADE,
    inicio_minutos     SMALLINT NOT NULL CHECK (inicio_minutos >= 0 AND inicio_minutos < 1440),
    fim_minutos        SMALLINT NOT NULL CHECK (fim_minutos > 0 AND fim_minutos <= 1440),
    CHECK (fim_minutos > inicio_minutos)
);

CREATE INDEX idx_date_exception_blocks_exception ON date_exception_blocks (date_exception_id);
