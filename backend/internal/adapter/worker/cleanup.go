package worker

import (
	"context"
	"log"
	"time"
)

// repositorioExpiravel é qualquer repositório de tokens que apaga os
// registros vencidos — cadastro pendente, recuperação de senha, pré-cadastro
// e cancelamento seguem esse contrato.
type repositorioExpiravel interface {
	RemoverExpirados() error
}

// CleanupWorker apaga periodicamente os tokens vencidos que ninguém mais vai
// consumir. As operações já fazem limpeza best-effort ao consumir um token
// (ex: ConfirmarCadastro chama RemoverExpirados no sucesso), mas rotas
// pouco usadas ou nunca concluídas deixariam lixo parado sem este worker —
// importante sobretudo para pré-cadastro e cancelamento, que carregam PII de
// convidados nunca confirmados.
type CleanupWorker struct {
	repos     []repositorioExpiravel
	intervalo time.Duration
}

// NovoCleanupWorker cria uma instância de CleanupWorker com os repositórios a
// limpar e o intervalo entre execuções.
func NovoCleanupWorker(intervalo time.Duration, repos ...repositorioExpiravel) *CleanupWorker {
	return &CleanupWorker{repos: repos, intervalo: intervalo}
}

// Executar roda o worker até ctx ser cancelado. Erros de execução são só
// logados — uma falha numa limpeza não deve derrubar o processo, a próxima
// tentativa acontece no próximo tick.
func (w *CleanupWorker) Executar(ctx context.Context) {
	w.limpar()

	ticker := time.NewTicker(w.intervalo)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.limpar()
		}
	}
}

func (w *CleanupWorker) limpar() {
	for _, repo := range w.repos {
		if err := repo.RemoverExpirados(); err != nil {
			log.Printf("worker de limpeza: erro ao remover tokens expirados: %v", err)
		}
	}
}
