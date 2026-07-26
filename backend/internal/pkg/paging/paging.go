// Package paging define a fatia pedida de uma listagem — quantos itens e a
// partir de qual posição. Existe para nenhuma consulta crescer sem limite
// junto com o banco: a vitrine, o painel de moderação e o histórico de
// agendamentos são listas que só aumentam, e uma delas sem teto acaba sendo o
// jeito mais barato de derrubar a API.
package paging

const (
	// LimitePadrao é quanto uma listagem devolve quando o cliente não pede
	// nada — o suficiente para uma tela cheia sem carregar o banco inteiro.
	LimitePadrao = 100
	// LimiteMaximo é o teto por resposta. Sem ele, ?limite=999999 traria a
	// tabela inteira e o limite não protegeria de nada.
	LimiteMaximo = 200
)

// Pagina é a fatia pedida de uma listagem.
type Pagina struct {
	Limite int
	Offset int
}

// Valida devolve a página já normalizada. Os repositórios chamam antes de
// montar o SQL: uma Pagina{} zerada que escapasse de um caller viraria
// `LIMIT 0` e devolveria uma lista vazia calada — o tipo de erro que só
// aparece em produção, com o usuário achando que perdeu os dados.
func (p Pagina) Valida() Pagina {
	return Normalizar(p.Limite, p.Offset)
}

// Normalizar devolve uma página válida a partir de valores crus (a query
// string, tipicamente): limite ausente ou negativo vira LimitePadrao, acima do
// teto vira LimiteMaximo, e offset negativo vira 0. Valor inválido não vira
// erro — vira o padrão, porque uma listagem não deve falhar por causa de um
// parâmetro mal escrito na URL.
func Normalizar(limite, offset int) Pagina {
	if limite <= 0 {
		limite = LimitePadrao
	}
	if limite > LimiteMaximo {
		limite = LimiteMaximo
	}
	if offset < 0 {
		offset = 0
	}
	return Pagina{Limite: limite, Offset: offset}
}
