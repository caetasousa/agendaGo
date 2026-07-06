SHELL := /bin/bash

.PHONY: test test-all test-backend test-backend-integration test-frontend test-e2e

# Roda os testes rápidos de backend e frontend (sem Docker, sem browsers).
# É o alvo padrão para checar o projeto antes de commitar.
test: test-backend test-frontend

# Roda toda a suíte: rápidos + integração (Testcontainers) + E2E (Playwright).
# Exige Docker rodando; para o E2E, também exige `docker compose up` no ar
# (API em :8080 e web em :5173) e o browser do Playwright instalado
# (`cd frontend && npx playwright install chromium`).
test-all: test-backend-integration test-frontend test-e2e

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
