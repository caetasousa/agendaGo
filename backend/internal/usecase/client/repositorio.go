package client

import (
	"time"

	"agendago/internal/domain/client"
	"agendago/internal/domain/precadastro"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/signup"
)

// repositorioClient persiste e consulta clientes, incluindo a conversão de um
// convidado em conta (preservando o ID e o histórico de agendamentos).
type repositorioClient interface {
	Salvar(c *client.Client) error
	BuscarPorEmail(email string) (*client.Client, error)
	BuscarPorID(id string) (*client.Client, error)
	ConverterEmConta(id, senhaHash, telefone string) error
}

// buscadorProvider verifica se o email já pertence a um prestador — o email é
// único entre clientes e prestadores.
type buscadorProvider interface {
	BuscarPorEmail(email string) (*provider.Provider, error)
}

// repositorioCadastroPendente persiste e consome os cadastros à espera de
// confirmação por email. Consumir apaga o registro ao lê-lo (uso único).
type repositorioCadastroPendente interface {
	Salvar(p *signup.Pendente) error
	Consumir(tokenHash string) (*signup.Pendente, error)
	RemoverPorEmail(email string) error
	RemoverExpirados() error
}

// enviadorCadastro envia os emails do fluxo de cadastro.
type enviadorCadastro interface {
	// EnviarConfirmacaoCadastro manda o link de confirmação para um email novo
	// ou de convidado.
	EnviarConfirmacaoCadastro(email, nome, token string, expiraEm time.Time)
	// EnviarAvisoContaExistente avisa que o email já tem conta (entre/recupere
	// a senha) — enviado no lugar do link, sem revelar isso na resposta HTTP.
	EnviarAvisoContaExistente(email, nome string)
}

// hasherSenha gera o hash da senha em texto puro para persistência.
type hasherSenha interface {
	Gerar(senha string) (string, error)
}

// repositorioPreCadastro consulta e consome os tokens de pré-cadastro gerados
// a partir do token de cancelamento do convidado. BuscarPorTokenHash só lê —
// serve o pré-preenchimento do formulário, que pode acontecer mais de uma
// vez. Consumir apaga o registro ao lê-lo (uso único de verdade), chamado só
// no submit final que materializa a conta.
type repositorioPreCadastro interface {
	BuscarPorTokenHash(tokenHash string) (*precadastro.PreCadastro, error)
	Consumir(tokenHash string) (*precadastro.PreCadastro, error)
	RemoverExpirados() error
}
