-- A tabela passa a guardar cadastros pendentes dos DOIS públicos: o prestador
-- também confirma o email antes de a conta existir. A coluna diz qual conta
-- criar quando o token for consumido.
--
-- NOT NULL direto, sem passo intermediário: o banco é recriado a cada mudança
-- de schema neste momento do projeto, então não há linha antiga para preencher
-- — e preencher exigiria decidir em SQL o que é decisão do domínio. Quando
-- existir base real, uma coluna obrigatória volta a precisar de três passos.
ALTER TABLE cadastros_pendentes ADD COLUMN tipo VARCHAR(20) NOT NULL;
