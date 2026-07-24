-- Estado (state) de uso único do fluxo OAuth, para proteção contra CSRF.
-- Guarda só o hash do valor entregue ao usuário, como em sessions. publico
-- (client ou provider) fica gravado aqui — verificado server-side no
-- callback junto do state, em vez de lido de um cookie separado sem vínculo
-- criptográfico com o state consumido.
CREATE TABLE oauth_states (
    state_hash CHAR(64)    PRIMARY KEY,
    provedor   VARCHAR(20) NOT NULL,
    publico    VARCHAR(10) NOT NULL,
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expira_em  TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_oauth_states_expira_em ON oauth_states (expira_em);
