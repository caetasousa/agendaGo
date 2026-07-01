# agendaGo

> API de agendamento entre clientes e prestadores de serviço, construída em Go com arquitetura hexagonal.

---

## Sumário

- [Requisitos](#requisitos)
- [Executando o projeto](#executando-o-projeto)
- [Rotas disponíveis](#rotas-disponíveis)
- [Testes](#testes)
- [Estrutura do projeto](#estrutura-do-projeto)
- [Regras de negócio](#regras-de-negócio)

---

## Requisitos

- [Docker](https://docs.docker.com/get-docker/) e Docker Compose

---

## Executando o projeto

```bash
docker compose up
```

Os serviços sobem na seguinte ordem:

```
Postgres  →  Flyway (migrations)  →  API (hot reload)
```

| Serviço | Descrição |
|---------|-----------|
| **Postgres** | Aguarda ficar saudável antes de permitir o próximo passo |
| **Flyway** | Aplica as migrations e encerra |
| **API** | Sobe com hot reload via Air — alterações em `.go` reiniciam a API automaticamente |

A API estará disponível em `http://localhost:8080`.

### Encerrando

```bash
docker compose down        # mantém os dados do banco
docker compose down -v     # apaga os dados do banco junto
```

---

## Rotas disponíveis

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/health` | Status do servidor |
| `POST` | `/providers` | Cadastrar prestador |
| `GET` | [`/swagger/index.html`](http://localhost:8080/swagger/index.html) | Documentação interativa |

---

## Testes

Os testes rodam localmente sem Docker, exigem apenas Go instalado:

```bash
go test ./test/... -v
```

```
test/
├── domain/    testes das regras de negócio
├── usecase/   testes dos fluxos de usecase
└── handler/   testes do contrato HTTP
```

---

## Estrutura do projeto

```
agendaGo/
├── cmd/api/              entrypoint do servidor HTTP
├── config/               configuração do servidor chi (porta, timeouts)
├── internal/
│   ├── adapter/
│   │   ├── http/
│   │   │   ├── dto/      request e response HTTP
│   │   │   ├── handler/  handlers das rotas
│   │   │   └── middleware/
│   │   └── repository/   implementações de repositório (memória, postgres)
│   ├── domain/
│   │   ├── appointment/  agendamento e máquina de estados
│   │   ├── availability/ disponibilidade do prestador
│   │   ├── client/       cliente
│   │   ├── provider/     prestador de serviço
│   │   └── slot/         slots de horário (calculados sob demanda)
│   └── usecase/
│       ├── appointment/  solicitar, confirmar, recusar, cancelar, expirar
│       ├── availability/ definir disponibilidade e exceções
│       ├── client/       cadastro e busca de cliente
│       └── provider/     cadastro e configuração do prestador
├── migrations/           arquivos SQL versionados pelo Flyway (V1__, V2__...)
└── test/
    ├── domain/
    ├── handler/
    └── usecase/
```

---

## Regras de negócio

Consulte [regra-de-negocio.md](regra-de-negocio.md) para entender o modelo completo:
disponibilidade do prestador, geração de slots, ciclo de vida do agendamento e decisões de arquitetura.
