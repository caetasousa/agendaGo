-- Identidade de login social (OIDC) vinculada a um usuário. sub é o
-- identificador estável do usuário no provedor (Google), único por provedor.
-- user_type distingue provider de client, como em sessions.
CREATE TABLE social_identities (
    id        UUID         PRIMARY KEY,
    provedor  VARCHAR(20)  NOT NULL,
    sub       VARCHAR(255) NOT NULL,
    user_id   UUID         NOT NULL,
    user_type VARCHAR(10)  NOT NULL,
    email     VARCHAR(255),
    criado_em TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (provedor, sub)
);

CREATE INDEX idx_social_identities_user ON social_identities (user_id, user_type);
