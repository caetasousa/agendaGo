-- Prestadores de serviço e os blocos do seu expediente padrão (configurável
-- em Preferências). Validações de negócio — faixas de minutos, fim > início,
-- granularidade — são responsabilidade do domínio, não do banco.
--
-- Nenhuma coluna de capacidade tem DEFAULT: com que valor uma agenda nasce é
-- decisão do domínio (`provider.Novo`), e todo INSERT escreve a coluna. Um
-- DEFAULT aqui seria uma segunda opinião sobre a mesma pergunta, capaz de
-- divergir da primeira sem que nada acuse.
CREATE TABLE providers (
    id                              UUID         PRIMARY KEY,
    nome                            VARCHAR(100) NOT NULL,
    email                           VARCHAR(255) NOT NULL UNIQUE,
    telefone                        VARCHAR(30)  NOT NULL,
    senha_hash                      VARCHAR(255) NOT NULL,
    ativo                           BOOLEAN      NOT NULL,
    aceita_agendamentos             BOOLEAN      NOT NULL,
    descanso_minutos                INT          NOT NULL,
    duracao_atendimento_minutos     INT          NOT NULL,
    permite_marcacao_pelo_prestador BOOLEAN      NOT NULL,
    criado_em                       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    atualizado_em                   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE horarios_padrao (
    id             UUID     PRIMARY KEY,
    provider_id    UUID     NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    inicio_minutos SMALLINT NOT NULL,
    fim_minutos    SMALLINT NOT NULL
);

CREATE INDEX idx_horarios_padrao_provider ON horarios_padrao (provider_id);
