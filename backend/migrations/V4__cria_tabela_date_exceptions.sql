-- Definições próprias de data do prestador: bloqueio (dia indisponível) ou
-- extra (horários personalizados que substituem o expediente padrão).
-- UNIQUE (provider_id, data) é unicidade técnica: uma definição por data,
-- base do upsert. Valores de tipo e faixas de minutos são validados no domínio.
CREATE TABLE date_exceptions (
    id          UUID        PRIMARY KEY,
    provider_id UUID        NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    data        DATE        NOT NULL,
    tipo        VARCHAR(10) NOT NULL,
    criado_em   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider_id, data)
);

CREATE INDEX idx_date_exceptions_provider_data ON date_exceptions (provider_id, data);

CREATE TABLE date_exception_blocks (
    id                 UUID     PRIMARY KEY,
    date_exception_id  UUID     NOT NULL REFERENCES date_exceptions(id) ON DELETE CASCADE,
    inicio_minutos     SMALLINT NOT NULL,
    fim_minutos        SMALLINT NOT NULL
);

CREATE INDEX idx_date_exception_blocks_exception ON date_exception_blocks (date_exception_id);
