package memoria

import "agendago/internal/pkg/paging"

// fatiar aplica limite e offset a uma lista já ordenada, como o LIMIT/OFFSET
// do Postgres: offset além do fim devolve lista vazia, nunca erro. Os
// repositórios de memória precisam do mesmo comportamento do repositório real,
// senão um teste de paginação passaria aqui e falharia no banco.
func fatiar[T any](itens []T, pag paging.Pagina) []T {
	pag = pag.Valida()
	if pag.Offset >= len(itens) {
		return nil
	}
	fim := pag.Offset + pag.Limite
	if fim > len(itens) {
		fim = len(itens)
	}
	return itens[pag.Offset:fim]
}
