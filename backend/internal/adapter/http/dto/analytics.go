package dto

// MetricasResponse é o resumo analítico do período de um prestador.
//
// PorStatus é um mapa (e não um campo por status) para acompanhar o ciclo de
// vida do agendamento sem migração de contrato: um status novo aparece como
// mais uma chave, e quem consome já recebe a lista completa — o backend sempre
// manda todos os estados, inclusive os zerados.
//
// As taxas são frações em [0,1] e podem vir nulas: null significa que não
// havia base para calcular (nenhum expediente ofertado, nenhum atendimento
// concluído), diferente de 0, que significa medida e igual a zero.
type MetricasResponse struct {
	De                 string         `json:"de"`
	Ate                string         `json:"ate"`
	PorStatus          map[string]int `json:"porStatus"`
	Total              int            `json:"total"`
	MinutosOfertados   int            `json:"minutosOfertados"`
	MinutosReservados  int            `json:"minutosReservados"`
	TaxaOcupacao       *float64       `json:"taxaOcupacao"`
	TaxaComparecimento *float64       `json:"taxaComparecimento"`
}
