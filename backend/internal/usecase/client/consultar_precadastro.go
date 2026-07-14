package client

import (
	"errors"
	"time"

	"agendago/internal/pkg/token"
)

// ErrPreCadastroInvalido é retornado quando o token de pré-cadastro não
// existe — genérico de propósito.
var ErrPreCadastroInvalido = errors.New("link inválido")

// ConsultarPreCadastroOutput contém os dados do convidado a pré-preencher no
// formulário de cadastro.
type ConsultarPreCadastroOutput struct {
	Nome     string
	Email    string
	Telefone string
}

// ConsultarPreCadastroUseCase resolve os dados de um convidado a partir do
// token de pré-cadastro recebido no email, para a tela de cadastro poder
// pré-preencher o formulário. Só lê — não consome o token, para o
// pré-preenchimento poder acontecer sem invalidar o submit final
// (ConcluirPreCadastroUseCase), que é quem de fato consome.
type ConsultarPreCadastroUseCase struct {
	preCadastro repositorioPreCadastro
}

// NovoConsultarPreCadastroUseCase cria uma instância de ConsultarPreCadastroUseCase com as dependências injetadas.
func NovoConsultarPreCadastroUseCase(preCadastro repositorioPreCadastro) *ConsultarPreCadastroUseCase {
	return &ConsultarPreCadastroUseCase{preCadastro: preCadastro}
}

// Executar devolve os dados do convidado a partir do token. Retorna
// ErrPreCadastroInvalido para token inexistente ou expirado.
func (uc *ConsultarPreCadastroUseCase) Executar(tokenPuro string) (*ConsultarPreCadastroOutput, error) {
	p, err := uc.preCadastro.BuscarPorTokenHash(token.Hash(tokenPuro))
	if err != nil {
		return nil, err
	}
	if p == nil || p.Expirado(time.Now()) {
		return nil, ErrPreCadastroInvalido
	}
	return &ConsultarPreCadastroOutput{Nome: p.Nome, Email: p.Email, Telefone: p.Telefone}, nil
}
