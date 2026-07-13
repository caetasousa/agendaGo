package dto

// SolicitarAgendamentoRequest é o pedido de reserva de um slot livre.
type SolicitarAgendamentoRequest struct {
	ProviderID    string `json:"providerId" validate:"required,uuid"`
	Data          string `json:"data" validate:"required,datetime=2006-01-02"`
	InicioMinutos int    `json:"inicioMinutos" validate:"min=0,max=1440"`
}

func (r SolicitarAgendamentoRequest) Validar() error {
	return validate.Struct(r)
}

// SolicitarConvidadoRequest é o pedido de reserva de um convidado sem cadastro.
// Além do slot, exige nome/email/telefone de contato.
type SolicitarConvidadoRequest struct {
	ProviderID    string `json:"providerId" validate:"required,uuid"`
	Data          string `json:"data" validate:"required,datetime=2006-01-02"`
	InicioMinutos int    `json:"inicioMinutos" validate:"min=0,max=1440"`
	Nome          string `json:"nome" validate:"required,min=2,max=100"`
	Email         string `json:"email" validate:"required,email"`
	Telefone      string `json:"telefone" validate:"required,min=8,max=30"`
}

func (r SolicitarConvidadoRequest) Validar() error {
	return validate.Struct(r)
}

// AgendamentoResponse é um agendamento pronto para exibição. O contato do
// cliente (email/telefone) permite ao prestador falar com quem agendou.
type AgendamentoResponse struct {
	ID              string `json:"id"`
	Data            string `json:"data"`
	InicioMinutos   int    `json:"inicioMinutos"`
	FimMinutos      int    `json:"fimMinutos"`
	Status          string `json:"status"`
	ExpiraEm        string `json:"expiraEm"`
	NomeCliente     string `json:"nomeCliente,omitempty"`
	EmailCliente    string `json:"emailCliente,omitempty"`
	TelefoneCliente string `json:"telefoneCliente,omitempty"`
	NomePrestador   string `json:"nomePrestador,omitempty"`
}

// ListarAgendamentosResponse contém os agendamentos do usuário autenticado.
type ListarAgendamentosResponse struct {
	Agendamentos []AgendamentoResponse `json:"agendamentos"`
}

// DetalheCancelamentoResponse descreve o agendamento apontado por um token de
// cancelamento, para a página pública de confirmação do convidado.
type DetalheCancelamentoResponse struct {
	NomePrestador string `json:"nomePrestador"`
	Data          string `json:"data"`
	InicioMinutos int    `json:"inicioMinutos"`
	FimMinutos    int    `json:"fimMinutos"`
	Status        string `json:"status"`
	PodeCancelar  bool   `json:"podeCancelar"`
}

// SlotDTO é um horário livre ofertável.
type SlotDTO struct {
	InicioMinutos int `json:"inicioMinutos"`
	FimMinutos    int `json:"fimMinutos"`
}

// DiaSlotsDTO são os slots livres de uma data.
type DiaSlotsDTO struct {
	Data  string    `json:"data"`
	Slots []SlotDTO `json:"slots"`
}

// SlotsResponse contém os slots livres do período consultado.
type SlotsResponse struct {
	Dias []DiaSlotsDTO `json:"dias"`
}

// PrestadorResumoDTO identifica um prestador na vitrine e no link público de
// agendamento.
type PrestadorResumoDTO struct {
	ID                        string `json:"id"`
	Nome                      string `json:"nome"`
	DuracaoAtendimentoMinutos int    `json:"duracaoAtendimentoMinutos"`
	AceitaAgendamentos        bool   `json:"aceitaAgendamentos"`
}

// ListarPrestadoresResponse contém os prestadores com agenda ativa.
type ListarPrestadoresResponse struct {
	Prestadores []PrestadorResumoDTO `json:"prestadores"`
}
