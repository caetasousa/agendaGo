SHELL := /bin/bash

.PHONY: test test-fast

# Roda todos os testes (rápidos + integração via Testcontainers) e mostra,
# de forma legível, cada caso que passou ou falhou. Exige Docker rodando.
# Falha o alvo (exit != 0) se qualquer teste quebrar, para não mascarar erros.
#
# `-race` igual ao CI: a API serve requisições concorrentes, e corrida de dados
# não quebra o teste sem o detector — só aparece em produção sob carga. Fica
# fora do `test-fast` de propósito, que existe para o ciclo curto.
test:
	@set -o pipefail; go test -race -tags=integration -v ./... 2>&1 \
		| grep -E -- '--- (PASS|FAIL)' \
		| sed -E 's/.*--- (PASS|FAIL): /\1 /; s/ \(.*\)//; s@/@ › @; s/_/ /g'

# Roda só os testes rápidos (em memória, sem Docker).
test-fast:
	@go test ./...
