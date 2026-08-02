-- Trilha de auditoria das ações sensíveis: quem fez o quê, com quem, quando.
--
-- Append-only por decisão, não por constraint: o repositório expõe apenas
-- inserção e leitura, sem UPDATE nem DELETE. O banco não impede um UPDATE
-- manual, e não é ele que deve impedir — mas nada na aplicação escreve por
-- cima de um registro já gravado.
--
-- `ator_id` NÃO tem foreign key, de propósito. A trilha precisa sobreviver ao
-- desaparecimento do ator: um cliente anonimizado, uma conta removida, um admin
-- que deixou de existir. Com FK e CASCADE, apagar a conta apagaria justamente o
-- registro de que ela foi apagada. Mesma razão para `alvo_id`.
--
-- `detalhe` é JSONB para o contexto variar por ação sem migration nova — o
-- motivo de um banimento não tem a mesma forma que o alvo de uma exclusão.
-- Nunca guarda dado pessoal: o alvo é identificado por id, não por nome ou
-- email, senão a trilha viraria a cópia do que a anonimização acabou de apagar.
CREATE TABLE auditoria (
    id         UUID        PRIMARY KEY,
    ator_id    UUID        NOT NULL,
    ator_tipo  VARCHAR(20) NOT NULL,
    acao       VARCHAR(60) NOT NULL,
    alvo_tipo  VARCHAR(20) NOT NULL,
    alvo_id    UUID        NOT NULL,
    detalhe    JSONB,
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- As duas perguntas que se faz a uma trilha: "o que aconteceu com este alvo?"
-- e "o que esta pessoa andou fazendo?".
CREATE INDEX idx_auditoria_alvo ON auditoria (alvo_tipo, alvo_id, criado_em DESC);
CREATE INDEX idx_auditoria_ator ON auditoria (ator_id, criado_em DESC);
