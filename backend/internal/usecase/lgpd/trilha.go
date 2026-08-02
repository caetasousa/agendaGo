package lgpd

import (
	"log/slog"

	"agendago/internal/domain/auditoria"

	"github.com/google/uuid"
)

// registrar grava a trilha sem deixar que uma falha nela derrube a operação
// principal. O ator é o próprio titular: estes dois usecases atendem pedidos
// feitos pela pessoa sobre os próprios dados.
//
// Nenhum dado pessoal vai no detalhe — o alvo é o id, e guardar nome ou email
// aqui recriaria na trilha o que a anonimização acabou de apagar.
func (uc *AnonimizarUseCase) registrar(clientID string) {
	gravar(uc.auditoria, clientID, auditoria.AcaoAnonimizarCliente)
}

func (uc *ExportarUseCase) registrar(clientID string) {
	gravar(uc.auditoria, clientID, auditoria.AcaoExportarDados)
}

func gravar(trilha registradorAuditoria, clientID string, acao auditoria.Acao) {
	if trilha == nil {
		return
	}
	reg, err := auditoria.Novo(uuid.NewString(), clientID, auditoria.TipoCliente, acao, auditoria.TipoCliente, clientID, nil)
	if err != nil {
		slog.Error("falha ao montar registro de auditoria", slog.String("erro", err.Error()))
		return
	}
	if err := trilha.Registrar(reg); err != nil {
		slog.Error("falha ao gravar auditoria", slog.String("erro", err.Error()), slog.String("acao", string(acao)))
	}
}
