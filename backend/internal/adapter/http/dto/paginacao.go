package dto

import "agendago/internal/pkg/paging"

// PaginacaoDTO acompanha toda listagem paginada: quantos itens existem no
// servidor e qual fatia esta resposta traz. É o que permite ao frontend saber
// se ainda há o que carregar sem adivinhar pelo tamanho da lista recebida.
//
// Embutido (sem tag json), os três campos aparecem no mesmo nível do corpo.
type PaginacaoDTO struct {
	Total  int `json:"total"`
	Limite int `json:"limite"`
	Offset int `json:"offset"`
}

// NovaPaginacao monta o bloco de paginação da resposta a partir da página
// pedida e do total apurado.
func NovaPaginacao(pag paging.Pagina, total int) PaginacaoDTO {
	return PaginacaoDTO{Total: total, Limite: pag.Limite, Offset: pag.Offset}
}
