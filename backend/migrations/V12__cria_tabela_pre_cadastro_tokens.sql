-- Tokens de pré-cadastro: carregam nome/email/telefone de um convidado do
-- email direto para a tela de cadastro, para pré-preencher o formulário sem
-- redigitar. Uso único (a leitura consome); sem expira_em, mesmo padrão de
-- cancelamento_tokens — a validade acompanha o token de cancelamento gerado
-- junto no mesmo email.
CREATE TABLE pre_cadastro_tokens (
    token_hash CHAR(64)    PRIMARY KEY,
    nome       VARCHAR(100) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    telefone   VARCHAR(30)  NOT NULL,
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
