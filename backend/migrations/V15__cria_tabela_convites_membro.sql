-- Convite para operar a agenda de outra pessoa. O dono informa só o email; o
-- nome, o telefone e a senha são preenchidos por quem aceita — por isso este
-- token não se parece com cadastros_pendentes (V9), que carrega os dados que a
-- própria pessoa já digitou no formulário.
--
-- Guarda apenas o hash do token, como os demais tokens do sistema: o valor em
-- texto puro existe só dentro do email enviado.
--
-- Sem FK para usuarios de propósito: no caso normal a pessoa convidada ainda
-- não tem conta, e é o aceite que a cria.
CREATE TABLE convites_membro (
    token_hash  CHAR(64)     PRIMARY KEY,
    email       VARCHAR(255) NOT NULL,
    provider_id UUID         NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    papel       VARCHAR(20)  NOT NULL,
    criado_em   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expira_em   TIMESTAMPTZ  NOT NULL
);

-- Um convite pendente por email e agenda: reconvidar substitui o anterior em
-- vez de acumular tokens válidos para o mesmo destino.
CREATE UNIQUE INDEX idx_convites_membro_email_provider
    ON convites_membro (email, provider_id);

-- A limpeza dos expirados varre por data.
CREATE INDEX idx_convites_membro_expira_em ON convites_membro (expira_em);
