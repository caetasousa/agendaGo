-- Equipe vira um recurso opcional da agenda, ligado em Configurações. Quem
-- trabalha sozinho — a maioria — não precisa ver a tela nem carregar a ideia
-- de convidar alguém.
--
-- Em três passos porque a coluna é NOT NULL e a tabela pode ter linhas: cria
-- nula, preenche e só então exige o preenchimento — mesmo estilo da V17.
ALTER TABLE providers ADD COLUMN permite_equipe BOOLEAN;

-- Migração de DADOS, não DEFAULT de regra de negócio: quem já convidou alguém
-- fica com o recurso ligado, porque desligá-lo sob os pés de quem já opera
-- esconderia gente com acesso à agenda. O resto nasce desligado — o mesmo
-- valor que o domínio dá a um prestador novo.
UPDATE providers p SET permite_equipe =
    EXISTS (SELECT 1 FROM provider_membros m WHERE m.provider_id = p.id AND m.papel <> 'dono')
    OR EXISTS (SELECT 1 FROM convites_membro c WHERE c.provider_id = p.id);

ALTER TABLE providers ALTER COLUMN permite_equipe SET NOT NULL;
