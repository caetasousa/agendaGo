# agendaGo

## Migrations (banco de dados)

O banco é um detalhe de infraestrutura — ele não conhece regras de negócio.

Constraints permitidas nas migrations:
- `NOT NULL` — o campo deve sempre existir
- `UNIQUE` — unicidade técnica (ex: email)
- `PRIMARY KEY`, `FOREIGN KEY` — relações entre tabelas
- Tipos corretos (`UUID`, `VARCHAR`, `TIMESTAMPTZ`)

Não usar nas migrations:
- `DEFAULT` com valores que representam regras de negócio — essa responsabilidade é do domínio
- `CHECK` constraints que validam regras de negócio — essa responsabilidade é do domínio

---

## README

Sempre que uma nova rota for criada, ela deve ser adicionada à tabela de rotas no `README.md`. A rota vira um link clicável apontando para a operação correspondente no Swagger UI: `http://localhost:8080/swagger/index.html#/{tag}/{método}_{caminho-com-underscore}` (ex.: `POST /auth/me` com tag `auth` → `#/auth/get_auth_me`). Para descobrir o id exato de uma operação nova, inspecionar os elementos `[id^="operations-"]` na página do Swagger renderizada.

### Estrutura de documentação

O `README.md` é enxuto — cartão de visitas: badges, stack resumida, quick start, tabela de rotas, comandos de teste, e links para `docs/`. Não crescer o README com conteúdo detalhado; extrair para arquivos dedicados em `docs/`:

- `docs/tecnologias.md` — guia de estudo do stack (o que é, por que está no projeto, fontes oficiais para aprofundar)
- `docs/testes.md` — instruções detalhadas de cada camada de teste (build tags, Testcontainers, Playwright)
- `docs/regra-de-negocio.md` — modelo de negócio

Ao adicionar uma tecnologia nova ou uma decisão de arquitetura relevante, documentar em `docs/tecnologias.md` seguindo o padrão já estabelecido (o que é → por que está aqui, com referência a arquivo real do código → fontes para estudo), não só mencionar em passant no README.

---

## Comentários no código

Comentários sempre em português.

- Identificadores exportados (Go) recebem doc comment no padrão `// Nome é/faz X`, descrevendo o comportamento — inclusive casos de erro relevantes (ex: `// Executar valida os dados, verifica duplicidade de email e persiste o novo prestador.`).
- Arquivo com papel não óbvio pelo nome/caminho ganha um comentário de cabeçalho explicando para que ele serve (ex: `// Cliente HTTP fino sobre fetch para falar com a API Go.`).
- Fora isso, comentário só quando o "porquê" não é óbvio pelo código — uma decisão, um trade-off, uma limitação do ambiente (ex: `// localStorage indisponível: mantém a escolha só nesta sessão`). Nunca comentário descrevendo o que o código já deixa claro sozinho.
- Anotações Swag/godoc (`@Summary`, `@Router` etc.) seguem o formato exigido pela ferramenta, não esse padrão.

---

## Convenção de commits

Cada commit deve ter um único contexto — nunca misturar feat, fix, docs, chore ou refactor no mesmo commit. Se uma tarefa envolve múltiplos contextos, separar em commits distintos.

Mensagens de commit seguem o padrão Conventional Commits, sempre em português:

- **feat** — nova funcionalidade
- **fix** — correção de bug
- **docs** — só documentação
- **chore** — tarefa de manutenção que não afeta o código de produção (configuração, build, `.gitignore`, dependências)
- **refactor** — reorganização de código sem mudar comportamento
- **test** — adição ou ajuste de testes
