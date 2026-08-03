-- Equipe vira um recurso opcional da agenda, ligado em Configurações. Quem
-- trabalha sozinho — a maioria — não precisa ver a tela nem carregar a ideia
-- de convidar alguém.
--
-- NOT NULL direto: com que valor uma agenda nasce com o recurso é decisão do
-- domínio (`provider.Novo`), e não há linha antiga para preencher. Derivar o
-- valor de quem já tem membro ou convite poria a decisão no SQL — irreversível,
-- porque migration aplicada não se corrige.
ALTER TABLE providers ADD COLUMN permite_equipe BOOLEAN NOT NULL;
