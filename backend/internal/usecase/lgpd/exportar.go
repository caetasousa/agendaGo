package lgpd

import (
	"agendago/internal/pkg/paging"
)

// ExportarUseCase entrega ao cliente os dados que o sistema guarda sobre ele.
type ExportarUseCase struct {
	clients      repositorioClient
	agendamentos repositorioAppointment
	auditoria    registradorAuditoria
}

// NovoExportarUseCase cria uma instância de ExportarUseCase com os repositórios injetados.
func NovoExportarUseCase(clients repositorioClient, agendamentos repositorioAppointment, trilha registradorAuditoria) *ExportarUseCase {
	return &ExportarUseCase{clients: clients, agendamentos: agendamentos, auditoria: trilha}
}

// Executar monta o pacote de portabilidade do cliente.
//
// Retorna ErrClienteNaoEncontrado quando a conta não existe. Conta já
// anonimizada exporta o que sobrou — que é justamente nada de pessoal —, em vez
// de recusar: recusar sugeriria que ainda há algo guardado ali.
func (uc *ExportarUseCase) Executar(clientID string) (*DadosExportados, error) {
	c, err := uc.clients.BuscarPorID(clientID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrClienteNaoEncontrado
	}

	pag := paging.Pagina{Limite: maxAgendamentosExportados, Offset: 0}
	agendamentos, total, err := uc.agendamentos.ListarPorCliente(clientID, pag)
	if err != nil {
		return nil, err
	}

	exportados := make([]AgendamentoExportado, 0, len(agendamentos))
	for _, a := range agendamentos {
		exportados = append(exportados, AgendamentoExportado{
			Data:          a.Data,
			InicioMinutos: a.InicioMinutos,
			FimMinutos:    a.FimMinutos,
			Status:        string(a.Status),
			CriadoEm:      a.CriadoEm,
		})
	}

	uc.registrar(clientID)

	return &DadosExportados{
		ID:             c.ID,
		Nome:           c.Nome,
		Email:          c.Email,
		Telefone:       c.Telefone,
		CriadoEm:       c.CriadoEm,
		Agendamentos:   exportados,
		TotalNoPeriodo: total,
		Truncado:       total > len(exportados),
	}, nil
}
