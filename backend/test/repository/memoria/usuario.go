package memoria

import (
	"sync"
	"time"

	"agendago/internal/domain/usuario"
)

type UsuarioMemoria struct {
	mu    sync.RWMutex
	dados map[string]*usuario.Usuario
}

func NovoUsuarioMemoria() *UsuarioMemoria {
	return &UsuarioMemoria{dados: make(map[string]*usuario.Usuario)}
}

func (r *UsuarioMemoria) Salvar(u *usuario.Usuario) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[u.ID] = u
	return nil
}

// BuscarPorEmail retorna (nil, nil) quando não há usuário com o email,
// seguindo o mesmo contrato do repositório Postgres.
func (r *UsuarioMemoria) BuscarPorEmail(email string) (*usuario.Usuario, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.dados {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

// BuscarPorID retorna (nil, nil) quando não há usuário com o id, seguindo o
// mesmo contrato do repositório Postgres.
func (r *UsuarioMemoria) BuscarPorID(id string) (*usuario.Usuario, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if u, ok := r.dados[id]; ok {
		return u, nil
	}
	return nil, nil
}

func (r *UsuarioMemoria) Atualizar(u *usuario.Usuario) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[u.ID] = u
	return nil
}

// AtualizarSenha troca o hash do usuário. Não faz nada quando o id não existe,
// como o UPDATE do Postgres, que não erra em zero linhas afetadas.
func (r *UsuarioMemoria) AtualizarSenha(id, senhaHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.dados[id]; ok {
		u.SenhaHash = senhaHash
		u.AtualizadoEm = time.Now()
	}
	return nil
}
