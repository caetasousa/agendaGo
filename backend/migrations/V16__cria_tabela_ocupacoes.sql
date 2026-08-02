-- Compromisso pessoal do prestador: um intervalo do dia que deixa de ser
-- ofertado, sem redefinir o expediente inteiro.
--
-- Não é date_exception nem appointment, e a distinção importa:
--   date_exception é binária por dia — bloqueio proíbe o dia todo, extra
--     substitui o expediente inteiro. Marcar "das 14h às 15h estou no médico"
--     ali obrigaria a redigitar o resto do dia.
--   appointment carrega cliente, máquina de estados, TTL e notificação —
--     nada disso se aplica a um compromisso do próprio prestador.
--
-- `origem` já nasce aqui, com `manual` como único valor em uso. Ela existe
-- porque este é o canal genérico de "intervalo ocupado que não é agendamento":
-- uma integração de calendário externo entraria como outra origem, escrevendo
-- nesta mesma tabela, sem tocar no cálculo de slots.
--
-- `origem_externa_id` guarda o identificador do evento no sistema de origem,
-- para reconciliar o que já foi importado. Nulo para origem manual.
--
-- Sem CHECK em `origem` nem nos minutos: quais origens existem e o que é um
-- intervalo válido (fim > início, dentro do dia, na granularidade) é decisão
-- do domínio, não do banco.
CREATE TABLE ocupacoes (
    id                UUID         PRIMARY KEY,
    provider_id       UUID         NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    data              DATE         NOT NULL,
    inicio_minutos    SMALLINT     NOT NULL,
    fim_minutos       SMALLINT     NOT NULL,
    titulo            VARCHAR(120),
    origem            VARCHAR(20)  NOT NULL,
    origem_externa_id VARCHAR(255),
    criado_em         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    atualizado_em     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Espelha idx_appointments_provider_data: toda consulta de ocupação é por
-- prestador dentro de um período, tanto na oferta de horários quanto na
-- verificação de conflito no momento da reserva.
CREATE INDEX idx_ocupacoes_provider_data ON ocupacoes (provider_id, data);
