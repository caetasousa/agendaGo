package memoria

import (
	"sort"
	"sync"
	"time"

	"agendago/internal/domain/ocupacao"
)

type OcupacaoMemoria struct {
	mu    sync.RWMutex
	dados map[string]*ocupacao.Ocupacao
}

func NovoOcupacaoMemoria() *OcupacaoMemoria {
	return &OcupacaoMemoria{dados: make(map[string]*ocupacao.Ocupacao)}
}

func (r *OcupacaoMemoria) Salvar(o *ocupacao.Ocupacao) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[o.ID] = o
	return nil
}

// ListarPorPeriodo devolve os compromissos entre duas datas (inclusive),
// ordenados por data e horário, como o repositório Postgres.
func (r *OcupacaoMemoria) ListarPorPeriodo(providerID string, de, ate time.Time) ([]*ocupacao.Ocupacao, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var achados []*ocupacao.Ocupacao
	for _, o := range r.dados {
		if o.ProviderID != providerID {
			continue
		}
		if o.Data.Before(de) || o.Data.After(ate) {
			continue
		}
		achados = append(achados, o)
	}
	sort.Slice(achados, func(i, j int) bool {
		if achados[i].Data.Equal(achados[j].Data) {
			return achados[i].InicioMinutos < achados[j].InicioMinutos
		}
		return achados[i].Data.Before(achados[j].Data)
	})
	return achados, nil
}

// BuscarPorID retorna (nil, nil) quando não há compromisso com o id, seguindo
// o mesmo contrato do repositório Postgres.
func (r *OcupacaoMemoria) BuscarPorID(id string) (*ocupacao.Ocupacao, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if o, ok := r.dados[id]; ok {
		return o, nil
	}
	return nil, nil
}

// Remover apaga o compromisso, espelhando o contrato do Postgres — que não
// erra quando o id não existe.
func (r *OcupacaoMemoria) Remover(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.dados, id)
	return nil
}
