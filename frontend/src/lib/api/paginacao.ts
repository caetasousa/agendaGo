// Paginação das listagens da API.
// Espelha backend/internal/adapter/http/dto/paginacao.go

// Paginacao acompanha toda listagem: `total` é quanto existe no servidor,
// `limite`/`offset` descrevem a fatia recebida. É comparando o tamanho
// acumulado com o total que a tela sabe se ainda há o que carregar.
export interface Paginacao {
	total: number;
	limite: number;
	offset: number;
}

// Pagina é o que o chamador pede. Omitir tudo aceita o padrão da API.
export interface Pagina {
	limite?: number;
	offset?: number;
}

// POR_PAGINA é quanto as telas com "Carregar mais" pedem por vez. Menor que o
// padrão da API (100): a primeira tela abre mais rápido e quem precisa de mais
// pede mais.
export const POR_PAGINA = 50;

// queryDaPagina monta o "?limite=&offset=" (ou string vazia, quando nada foi
// pedido) para concatenar no caminho da requisição.
export function queryDaPagina(pagina?: Pagina): string {
	const params = new URLSearchParams();
	if (pagina?.limite !== undefined) params.set('limite', String(pagina.limite));
	if (pagina?.offset !== undefined) params.set('offset', String(pagina.offset));
	const query = params.toString();
	return query ? `?${query}` : '';
}
