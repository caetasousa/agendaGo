// Package worker contém as tarefas de fundo do sistema, disparadas por
// ticker em vez de por requisição HTTP.
package worker

import (
	"context"
	"log/slog"
	"time"
)

// lembrarUseCase é o subconjunto do usecase.LembrarUseCase usado pelo worker.
type lembrarUseCase interface {
	Executar(agora time.Time) error
}

// ReminderWorker dispara periodicamente o envio de lembretes de agendamentos
// confirmados. Roda uma vez no boot (facilita verificar manualmente reiniciando
// a API) e depois a cada intervalo, até o contexto ser cancelado.
type ReminderWorker struct {
	lembrar   lembrarUseCase
	intervalo time.Duration
}

// NovoReminderWorker cria uma instância de ReminderWorker com as dependências injetadas.
func NovoReminderWorker(lembrar lembrarUseCase, intervalo time.Duration) *ReminderWorker {
	return &ReminderWorker{lembrar: lembrar, intervalo: intervalo}
}

// Executar roda o worker até ctx ser cancelado. Erros de execução são só
// logados — uma falha numa checagem não deve derrubar o processo, a próxima
// tentativa acontece no próximo tick.
func (w *ReminderWorker) Executar(ctx context.Context) {
	w.checar()

	ticker := time.NewTicker(w.intervalo)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checar()
		}
	}
}

func (w *ReminderWorker) checar() {
	if err := w.lembrar.Executar(time.Now()); err != nil {
		slog.Error("worker de lembrete: erro ao checar agendamentos", slog.String("erro", err.Error()))
	}
}
