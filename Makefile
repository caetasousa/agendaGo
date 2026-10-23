SHELL := /bin/bash

.PHONY: test test-all checar-migrations test-backend test-backend-integration test-frontend test-e2e test-rollback

# Roda os testes rápidos de backend e frontend (sem Docker, sem browsers).
# É o alvo padrão para checar o projeto antes de commitar.
test: checar-migrations test-backend test-frontend

# Roda toda a suíte: rápidos + integração (Testcontainers) + E2E (Playwright).
# Exige Docker rodando; para o E2E, também exige `docker compose up` no ar
# (API em :8080 e web em :5173) e o browser do Playwright instalado
# (`cd frontend && npx playwright install chromium`).
test-all: test-backend-integration test-frontend test-e2e test-rollback

# Migration não escreve dado: o backfill é decisão do domínio, e em SQL vira uma
# segunda fonte da verdade sem teste nem conserto (ver CLAUDE.md). Sem esteira,
# é aqui que a regra reprova em vez de ficar só no texto.
checar-migrations:
	@if grep -nE '^[[:space:]]*(UPDATE|INSERT|DELETE)[[:space:]]' backend/migrations/*.sql; then \
		echo "migration com statement de dados — o preenchimento roda pelo domínio, não em SQL (ver CLAUDE.md)"; \
		exit 1; \
	fi

# Testes rápidos do backend (domínio, usecases, handlers — em memória).
test-backend:
	@$(MAKE) -C backend test-fast

# Backend completo: rápidos + integração via Testcontainers (exige Docker).
test-backend-integration:
	@$(MAKE) -C backend test

# Testes unitários do frontend (Vitest).
test-frontend:
	@cd frontend && npm run test:unit

# Testes E2E do frontend (Playwright). Exige `docker compose up` no ar.
test-e2e:
	@cd frontend && npm run test:e2e

# Ensaia o ciclo de rollback (backup carimbado, restore e o guard que impede
# voltar código para trás do schema) num ambiente descartável. Exige Docker;
# não toca em produção nem no banco de desenvolvimento.
test-rollback:
	@./scripts/testar-rollback.sh
