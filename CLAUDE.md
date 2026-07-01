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

Sempre que uma nova rota for criada, ela deve ser adicionada à tabela de rotas no `README.md`.

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
