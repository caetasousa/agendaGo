// DTOs dos direitos de portabilidade e exclusão.
package dto

// AgendamentoExportadoDTO é um agendamento no pacote de dados do cliente.
type AgendamentoExportadoDTO struct {
	Data          string `json:"data"`
	InicioMinutos int    `json:"inicioMinutos"`
	FimMinutos    int    `json:"fimMinutos"`
	Status        string `json:"status"`
	CriadoEm      string `json:"criadoEm"`
}

// DadosExportadosResponse é o pacote de portabilidade, servido como JSON para
// download.
type DadosExportadosResponse struct {
	ID           string                    `json:"id"`
	Nome         string                    `json:"nome"`
	Email        string                    `json:"email"`
	Telefone     string                    `json:"telefone,omitempty"`
	CriadoEm     string                    `json:"criadoEm"`
	Agendamentos []AgendamentoExportadoDTO `json:"agendamentos"`
	// Total é quantos agendamentos existem ao todo; Truncado avisa que a lista
	// acima não os traz todos. Sem isso a pessoa acharia que recebeu tudo.
	Total    int  `json:"total"`
	Truncado bool `json:"truncado"`
}
