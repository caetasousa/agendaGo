package memoria

import (
	"sync"

	"agendago/internal/domain/auditoria"
)

// AuditoriaMemoria espelha o repositório Postgres, inclusive na ausência: não
// há como atualizar nem remover um registro, porque a trilha é append-only.
type AuditoriaMemoria struct {
	mu        sync.RWMutex
	registros []*auditoria.Registro
}

func NovoAuditoriaMemoria() *AuditoriaMemoria {
	return &AuditoriaMemoria{}
}

func (r *AuditoriaMemoria) Registrar(reg *auditoria.Registro) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registros = append(r.registros, reg)
	return nil
}

// ListarPorAlvo devolve a trilha de um alvo, da mais recente para a mais
// antiga, como o Postgres ordena.
func (r *AuditoriaMemoria) ListarPorAlvo(alvoTipo, alvoID string, limite int) ([]*auditoria.Registro, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var achados []*auditoria.Registro
	for i := len(r.registros) - 1; i >= 0 && len(achados) < limite; i-- {
		if r.registros[i].AlvoTipo == alvoTipo && r.registros[i].AlvoID == alvoID {
			achados = append(achados, r.registros[i])
		}
	}
	return achados, nil
}

// Todos devolve a trilha inteira — só para os testes inspecionarem o que foi
// gravado, sem precisar conhecer alvo nem limite.
func (r *AuditoriaMemoria) Todos() []*auditoria.Registro {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copia := make([]*auditoria.Registro, len(r.registros))
	copy(copia, r.registros)
	return copia
}
