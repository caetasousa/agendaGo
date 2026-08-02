-- Link público legível: /agendar/joao-barbeiro no lugar de /agendar/{uuid}.
--
-- Em três passos porque a coluna é NOT NULL e a tabela pode ter linhas: cria
-- nula, preenche a partir do nome e só então exige o preenchimento — mesmo
-- estilo da V13.
--
-- O UPDATE é migração de DADOS, não DEFAULT de regra de negócio: quem gera o
-- slug de um prestador novo é o domínio, e é lá que moram as palavras
-- reservadas e o formato aceito.
ALTER TABLE providers ADD COLUMN slug VARCHAR(60);

-- Normalização em SQL para não depender da aplicação numa migração:
--   unaccent não está instalado, então a troca de acentuadas é explícita;
--   o que sobrar fora de [a-z0-9] vira hífen, hífens repetidos colapsam, e
--   os das pontas caem.
UPDATE providers SET slug = trim(both '-' from regexp_replace(
    regexp_replace(
        lower(translate(nome,
            'áàâãäéèêëíìîïóòôõöúùûüçÁÀÂÃÄÉÈÊËÍÌÎÏÓÒÔÕÖÚÙÛÜÇ',
            'aaaaaeeeeiiiiooooouuuucAAAAAEEEEIIIIOOOOOUUUUC')),
        '[^a-z0-9]+', '-', 'g'),
    '-+', '-', 'g'))
WHERE slug IS NULL;

-- Nome que normaliza para vazio (só símbolos, por exemplo) não pode virar slug
-- vazio: cai para um identificador derivado do id, que é único por construção.
UPDATE providers SET slug = 'prestador-' || substring(id::text, 1, 8)
WHERE slug IS NULL OR slug = '';

-- Desempate de homônimos: quem chegou primeiro (criado_em) fica com o slug
-- limpo, os seguintes recebem sufixo numérico.
WITH numerados AS (
    SELECT id, slug,
           row_number() OVER (PARTITION BY slug ORDER BY criado_em, id) AS posicao
    FROM providers
)
UPDATE providers p SET slug = p.slug || '-' || n.posicao
FROM numerados n
WHERE p.id = n.id AND n.posicao > 1;

ALTER TABLE providers ALTER COLUMN slug SET NOT NULL;

-- Unicidade é técnica, não regra de negócio: dois prestadores com o mesmo slug
-- tornariam o link público ambíguo.
ALTER TABLE providers ADD CONSTRAINT providers_slug_unico UNIQUE (slug);
