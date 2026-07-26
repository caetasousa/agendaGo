//go:build integration

package repository_test

import "agendago/internal/pkg/paging"

// paginaPadrao é a página que os testes usam quando a paginação não é o objeto
// do teste: o padrão da API (100 itens), folgado para os poucos registros que
// cada caso semeia.
var paginaPadrao = paging.Normalizar(0, 0)
