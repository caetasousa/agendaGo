package repository

import (
	"sync"

	"agendago/internal/domain/client"
)

type ClientMemoria struct {
	mu    sync.RWMutex
	dados map[string]*client.Client
}

func NovoClientMemoria() *ClientMemoria {
	return &ClientMemoria{dados: make(map[string]*client.Client)}
}

func (r *ClientMemoria) Salvar(c *client.Client) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[c.ID] = c
	return nil
}

// Atualizar espelha o contrato do Postgres; como BuscarPorID devolve o
// ponteiro guardado, reatribuir ao mapa mantém o estado consistente.
func (r *ClientMemoria) Atualizar(c *client.Client) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dados[c.ID] = c
	return nil
}

// AtualizarSenha persiste um novo hash de senha, espelhando o contrato do Postgres.
func (r *ClientMemoria) AtualizarSenha(id, senhaHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.dados[id]; ok {
		c.SenhaHash = senhaHash
	}
	return nil
}

// ConverterEmConta define senha e telefone num convidado sem trocar o ID,
// espelhando o contrato do Postgres.
func (r *ClientMemoria) ConverterEmConta(id, senhaHash, telefone string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.dados[id]; ok {
		c.SenhaHash = senhaHash
		c.Telefone = telefone
	}
	return nil
}

// Listar devolve os clientes com conta, para o painel de moderação do admin.
func (r *ClientMemoria) Listar() ([]*client.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var todos []*client.Client
	for _, c := range r.dados {
		if c.TemConta() {
			todos = append(todos, c)
		}
	}
	return todos, nil
}

// BuscarPorEmail retorna (nil, nil) quando não há cliente com o email,
// seguindo o mesmo contrato do repositório Postgres — inclusive para email
// vazio: no banco a coluna vira NULL e `email = ''` nunca casa, então aqui
// também não (clientes só-telefone não são encontráveis por email).
func (r *ClientMemoria) BuscarPorEmail(email string) (*client.Client, error) {
	if email == "" {
		return nil, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.dados {
		if c.Email == email {
			return c, nil
		}
	}
	return nil, nil
}

// BuscarPorID retorna (nil, nil) quando não há cliente com o id, seguindo
// o mesmo contrato do repositório Postgres.
func (r *ClientMemoria) BuscarPorID(id string) (*client.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if c, ok := r.dados[id]; ok {
		return c, nil
	}
	return nil, nil
}
