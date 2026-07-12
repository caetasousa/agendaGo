// Package admin contém os usecases de moderação: semeadura do admin no boot,
// listagem e banimento/reativação de prestadores e clientes.
package admin

import (
	domadmin "agendago/internal/domain/admin"

	"github.com/google/uuid"
)

// repositorioAdmin persiste e busca o administrador.
type repositorioAdmin interface {
	Salvar(a *domadmin.Admin) error
	BuscarPorEmail(email string) (*domadmin.Admin, error)
}

// hasherSenha gera o hash da senha em texto puro para persistência.
type hasherSenha interface {
	Gerar(senha string) (string, error)
}

// SemearUseCase garante que o admin configurado por variáveis de ambiente
// exista no banco. É idempotente: rodar de novo atualiza o hash da senha
// (útil para trocar a senha do admin sem mexer no banco).
type SemearUseCase struct {
	repo   repositorioAdmin
	hasher hasherSenha
}

// NovoSemearUseCase cria uma instância de SemearUseCase com as dependências injetadas.
func NovoSemearUseCase(repo repositorioAdmin, hasher hasherSenha) *SemearUseCase {
	return &SemearUseCase{repo: repo, hasher: hasher}
}

// Executar cria (ou atualiza) o admin com o email e a senha informados.
// Chamado no boot; email/senha vazios são ignorados (sistema sem admin).
func (uc *SemearUseCase) Executar(email, senha string) error {
	if email == "" || senha == "" {
		return nil
	}

	// Preserva o id de um admin já existente, para o upsert por email não
	// gerar linhas órfãs; o Salvar do repositório resolve o conflito.
	id := uuid.NewString()
	if existente, err := uc.repo.BuscarPorEmail(email); err != nil {
		return err
	} else if existente != nil {
		id = existente.ID
	}

	hash, err := uc.hasher.Gerar(senha)
	if err != nil {
		return err
	}

	a, err := domadmin.Novo(id, email, hash)
	if err != nil {
		return err
	}
	return uc.repo.Salvar(a)
}
