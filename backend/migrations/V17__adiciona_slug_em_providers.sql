-- Link público legível: /agendar/joao-barbeiro no lugar de /agendar/{uuid}.
--
-- NOT NULL direto, sem preencher linha nenhuma: gerar slug aqui exigiria
-- reimplementar em SQL o que `provider.GerarSlug` já faz — dobra de acentos,
-- formato aceito, desempate de homônimos — e a cópia de cá não teria teste nem
-- conserto, porque migration aplicada não se corrige. Quem gera slug é o
-- domínio, no cadastro e na renomeação.
ALTER TABLE providers ADD COLUMN slug VARCHAR(60) NOT NULL;

-- Unicidade é técnica, não regra de negócio: dois prestadores com o mesmo slug
-- tornariam o link público ambíguo.
ALTER TABLE providers ADD CONSTRAINT providers_slug_unico UNIQUE (slug);
