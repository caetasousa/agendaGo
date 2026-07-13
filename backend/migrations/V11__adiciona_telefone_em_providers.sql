-- Telefone de contato do prestador, coletado no cadastro e editável nas
-- Preferências. NOT NULL: todo prestador tem telefone (a validação leve de
-- formato fica no domínio).
ALTER TABLE providers ADD COLUMN telefone VARCHAR(30) NOT NULL;
