package handler

import (
	"net/http"
	"strconv"

	"agendago/internal/pkg/paging"
)

// paginaDaQuery lê ?limite= e ?offset= da requisição. Valor ausente, não
// numérico ou fora da faixa cai no padrão em vez de virar 400: uma listagem não
// precisa recusar a requisição por causa de um parâmetro mal escrito na URL, e
// o teto de paging garante que o pedido nunca vira "traga a tabela inteira".
func paginaDaQuery(r *http.Request) paging.Pagina {
	return paging.Normalizar(inteiroDaQuery(r, "limite"), inteiroDaQuery(r, "offset"))
}

func inteiroDaQuery(r *http.Request, nome string) int {
	n, err := strconv.Atoi(r.URL.Query().Get(nome))
	if err != nil {
		return 0
	}
	return n
}
