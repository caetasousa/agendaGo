package memoria

import (
	"sort"
	"sync"

	"agendago/internal/domain/membro"
	"agendago/internal/domain/usuario"
)

type MembroMemoria struct {
	mu    sync.RWMutex
	dados map[string]*membro.Membro
	// usuarios responde se o dono está banido — no Postgres isso é um join,
	// aqui é uma referência ao outro fake. Pode ser nil nos testes que não
	// exercitam DonoAtivo.
	usuarios *UsuarioMemoria
}

func NovoMembroMemoria() *MembroMemoria {
	return &MembroMemoria{dados: make(map[string]*membro.Membro)}
}

// NovoMembroMemoriaCom liga o fake de vínculos ao de usuários, necessário para
// DonoAtivo responder como o join do Postgres responderia.
func NovoMembroMemoriaCom(usuarios *UsuarioMemoria) *MembroMemoria {
	return &MembroMemoria{dados: make(map[string]*membro.Membro), usuarios: usuarios}
}

func (r *MembroMemoria) Salvar(m *membro.Membro) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[m.ID] = m
	return nil
}

// BuscarPorUsuario devolve o vínculo do usuário, ou (nil, nil) quando ele não
// tem nenhum — mesmo contrato do repositório Postgres. Ordena por criação para
// que a escolha não dependa da iteração do mapa, que em Go é aleatória.
func (r *MembroMemoria) BuscarPorUsuario(usuarioID string) (*membro.Membro, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var achados []*membro.Membro
	for _, m := range r.dados {
		if m.UsuarioID == usuarioID {
			achados = append(achados, m)
		}
	}
	if len(achados) == 0 {
		return nil, nil
	}
	ordenarPorCriacao(achados)
	return achados[0], nil
}

// DonoAtivo informa se a agenda tem dono ativo, seguindo o mesmo contrato do
// repositório Postgres. Precisa consultar os usuários porque o banimento mora
// lá — por isso o fake recebe o de usuários na construção.
func (r *MembroMemoria) DonoAtivo(providerID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, m := range r.dados {
		if m.ProviderID != providerID || m.Papel != membro.PapelDono {
			continue
		}
		if r.usuarios == nil {
			return false, nil
		}
		u, err := r.usuarios.BuscarPorID(m.UsuarioID)
		if err != nil {
			return false, err
		}
		return u != nil && u.Ativo, nil
	}
	return false, nil
}

// DonoDe devolve a conta dona da agenda, ou (nil, nil) quando não há dono —
// mesmo contrato do repositório Postgres.
func (r *MembroMemoria) DonoDe(providerID string) (*usuario.Usuario, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, m := range r.dados {
		if m.ProviderID != providerID || m.Papel != membro.PapelDono {
			continue
		}
		if r.usuarios == nil {
			return nil, nil
		}
		return r.usuarios.BuscarPorID(m.UsuarioID)
	}
	return nil, nil
}

// EmailDoDono devolve o email do dono da agenda, ou string vazia quando não
// há dono — mesmo contrato do repositório Postgres.
func (r *MembroMemoria) EmailDoDono(providerID string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, m := range r.dados {
		if m.ProviderID != providerID || m.Papel != membro.PapelDono {
			continue
		}
		if r.usuarios == nil {
			return "", nil
		}
		u, err := r.usuarios.BuscarPorID(m.UsuarioID)
		if err != nil {
			return "", err
		}
		if u == nil {
			return "", nil
		}
		return u.Email, nil
	}
	return "", nil
}

// Remover apaga um vínculo, espelhando o contrato do Postgres — que não erra
// quando o id não existe.
func (r *MembroMemoria) Remover(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.dados, id)
	return nil
}

// ListarPorProvider devolve todos os vínculos de uma agenda, ordenados por
// criação.
func (r *MembroMemoria) ListarPorProvider(providerID string) ([]*membro.Membro, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var membros []*membro.Membro
	for _, m := range r.dados {
		if m.ProviderID == providerID {
			membros = append(membros, m)
		}
	}
	ordenarPorCriacao(membros)
	return membros, nil
}

func ordenarPorCriacao(membros []*membro.Membro) {
	sort.Slice(membros, func(i, j int) bool {
		if membros[i].CriadoEm.Equal(membros[j].CriadoEm) {
			return membros[i].ID < membros[j].ID
		}
		return membros[i].CriadoEm.Before(membros[j].CriadoEm)
	})
}
