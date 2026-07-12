package dto

// UsuarioModeracaoDTO descreve um prestador ou cliente no painel do admin.
type UsuarioModeracaoDTO struct {
	ID                 string `json:"id"`
	Nome               string `json:"nome"`
	Email              string `json:"email"`
	Ativo              bool   `json:"ativo"`
	AceitaAgendamentos bool   `json:"aceitaAgendamentos"`
}

// ListarUsuariosResponse contém os usuários de um tipo (prestadores ou clientes).
type ListarUsuariosResponse struct {
	Usuarios []UsuarioModeracaoDTO `json:"usuarios"`
}

// AgendamentoAdminDTO é um agendamento na visão de detalhe do admin.
type AgendamentoAdminDTO struct {
	ID              string `json:"id"`
	Data            string `json:"data"`
	InicioMinutos   int    `json:"inicioMinutos"`
	FimMinutos      int    `json:"fimMinutos"`
	Status          string `json:"status"`
	NomeCliente     string `json:"nomeCliente,omitempty"`
	EmailCliente    string `json:"emailCliente,omitempty"`
	TelefoneCliente string `json:"telefoneCliente,omitempty"`
	NomePrestador   string `json:"nomePrestador,omitempty"`
}

// DetalhePrestadorResponse é o detalhe em leitura de um prestador.
type DetalhePrestadorResponse struct {
	ID                 string                `json:"id"`
	Nome               string                `json:"nome"`
	Email              string                `json:"email"`
	Ativo              bool                  `json:"ativo"`
	AceitaAgendamentos bool                  `json:"aceitaAgendamentos"`
	DescansoMinutos    int                   `json:"descansoMinutos"`
	DuracaoMinutos     int                   `json:"duracaoAtendimentoMinutos"`
	Agendamentos       []AgendamentoAdminDTO `json:"agendamentos"`
}

// DetalheClienteResponse é o detalhe em leitura de um cliente.
type DetalheClienteResponse struct {
	ID           string                `json:"id"`
	Nome         string                `json:"nome"`
	Email        string                `json:"email"`
	Telefone     string                `json:"telefone,omitempty"`
	Ativo        bool                  `json:"ativo"`
	TemConta     bool                  `json:"temConta"`
	Agendamentos []AgendamentoAdminDTO `json:"agendamentos"`
}
