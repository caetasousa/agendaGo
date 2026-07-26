-- A tabela passa a guardar cadastros pendentes dos DOIS públicos: o prestador
-- também confirma o email antes de a conta existir. A coluna diz qual conta
-- criar quando o token for consumido.
--
-- Em três passos porque a coluna é NOT NULL e a tabela pode ter linhas: cria
-- nula, preenche o que já existe (todo pendente de hoje é de cliente — era o
-- único fluxo com confirmação por email) e só então exige o preenchimento.
-- O UPDATE é migração de DADOS, não um DEFAULT de regra de negócio: quem
-- decide o tipo de cada novo cadastro é o domínio.
ALTER TABLE cadastros_pendentes ADD COLUMN tipo VARCHAR(20);

UPDATE cadastros_pendentes SET tipo = 'client' WHERE tipo IS NULL;

ALTER TABLE cadastros_pendentes ALTER COLUMN tipo SET NOT NULL;
