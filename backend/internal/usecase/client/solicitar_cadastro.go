// Package client contém os usecases de cadastro de cliente com verificação por
// email: solicitar o cadastro (gera token e envia email) e confirmá-lo (cria a
// conta ou converte um convidado, preservando o histórico de agendamentos).
package client

import (
	"errors"
	"time"

	"agendago/internal/domain/signup"
	"agendago/internal/pkg/token"
)

// ErrCadastroInvalido é retornado quando o token de confirmação não existe ou
// expirou — genérico de propósito.
var ErrCadastroInvalido = errors.New("cadastro inválido ou expirado")

// TTLConfirmacaoCadastro é o prazo para confirmar um cadastro a partir da
// solicitação.
const TTLConfirmacaoCadastro = 24 * time.Hour

// SolicitarCadastroInput contém os dados do cadastro a confirmar por email.
type SolicitarCadastroInput struct {
	Nome     string
	Email    string
	Telefone string
	Senha    string
}

// SolicitarCadastroUseCase inicia o cadastro de um cliente: guarda os dados
// pendentes (senha já hasheada) e envia o email de confirmação. Não cria a
// conta agora — ela só nasce quando a pessoa prova posse do email.
type SolicitarCadastroUseCase struct {
	clients   repositorioClient
	providers buscadorProvider
	pendentes repositorioCadastroPendente
	enviador  enviadorCadastro
	hasher    hasherSenha
}

// NovoSolicitarCadastroUseCase cria uma instância de SolicitarCadastroUseCase com as dependências injetadas.
func NovoSolicitarCadastroUseCase(clients repositorioClient, providers buscadorProvider, pendentes repositorioCadastroPendente, enviador enviadorCadastro, hasher hasherSenha) *SolicitarCadastroUseCase {
	return &SolicitarCadastroUseCase{clients: clients, providers: providers, pendentes: pendentes, enviador: enviador, hasher: hasher}
}

// Executar processa a solicitação de cadastro. A resposta é sempre a mesma
// (retorna nil sem sinalizar o caso) para não revelar se o email já existe:
//
//   - Email já é conta ativa: envia o aviso "você já tem conta" e não cria pendente.
//   - Convidado banido: nada acontece (banimento não é revertido por cadastro).
//   - Email novo ou convidado ativo: gera token, guarda o cadastro pendente e
//     envia o link de confirmação.
//
// A senha é hasheada em todos os caminhos, para equalizar o custo/timing e
// nunca guardar texto puro.
func (uc *SolicitarCadastroUseCase) Executar(in SolicitarCadastroInput) error {
	senhaHash, err := uc.hasher.Gerar(in.Senha)
	if err != nil {
		return err
	}

	// o email não pode já pertencer a um prestador (o email é único no sistema):
	// avisa que já há conta, sem criar cadastro pendente
	prestador, err := uc.providers.BuscarPorEmail(in.Email)
	if err != nil {
		return err
	}
	if prestador != nil {
		uc.enviador.EnviarAvisoContaExistente(in.Email, prestador.Nome)
		return nil
	}

	existente, err := uc.clients.BuscarPorEmail(in.Email)
	if err != nil {
		return err
	}
	if existente != nil {
		if existente.TemConta() {
			uc.enviador.EnviarAvisoContaExistente(in.Email, existente.Nome)
			return nil
		}
		if !existente.Ativo {
			// convidado banido: não deixa virar conta ativa pelo cadastro
			return nil
		}
	}

	// invalida pendentes anteriores do mesmo email: só o último link vale
	if err := uc.pendentes.RemoverPorEmail(in.Email); err != nil {
		return err
	}

	t, err := token.Gerar()
	if err != nil {
		return err
	}
	pendente := signup.Novo(token.Hash(t), in.Nome, in.Email, in.Telefone, senhaHash, TTLConfirmacaoCadastro)
	if err := uc.pendentes.Salvar(pendente); err != nil {
		return err
	}

	uc.enviador.EnviarConfirmacaoCadastro(in.Email, in.Nome, t, pendente.ExpiraEm)
	return nil
}
