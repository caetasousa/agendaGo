// DTOs dos compromissos pessoais do prestador — o intervalo do dia que ele
// reserva para si e que some da oferta de horários.
package dto

// CriarOcupacaoRequest é o corpo para registrar um compromisso pessoal. O
// prestador não vai no corpo: vem da sessão.
type CriarOcupacaoRequest struct {
	Data          string `json:"data" validate:"required,datetime=2006-01-02"`
	InicioMinutos int    `json:"inicioMinutos" validate:"min=0,max=1439"`
	FimMinutos    int    `json:"fimMinutos" validate:"required,min=1,max=1440"`
	Titulo        string `json:"titulo" validate:"max=120"`
}

// Validar aplica as regras declarativas do corpo. Fim maior que início,
// granularidade e demais invariantes do intervalo são do domínio.
func (r CriarOcupacaoRequest) Validar() error {
	return validate.Struct(r)
}

// OcupacaoResponse descreve um compromisso pessoal. Nunca é exposto a cliente:
// o título é lembrete do prestador, e para quem agenda o horário simplesmente
// não aparece.
type OcupacaoResponse struct {
	ID            string `json:"id"`
	Data          string `json:"data"`
	InicioMinutos int    `json:"inicioMinutos"`
	FimMinutos    int    `json:"fimMinutos"`
	Titulo        string `json:"titulo,omitempty"`
	Origem        string `json:"origem"`
}

// OcupacoesResponse é a lista de compromissos de um período.
type OcupacoesResponse struct {
	Ocupacoes []OcupacaoResponse `json:"ocupacoes"`
}
