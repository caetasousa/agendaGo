package client

import "agendago/internal/pkg/token"

// ConcluirPreCadastroInput contém o token de pré-cadastro e a senha escolhida.
type ConcluirPreCadastroInput struct {
	Token string
	Senha string
}

// ConcluirPreCadastroOutput contém os dados da conta criada/convertida.
type ConcluirPreCadastroOutput struct {
	ID    string
	Nome  string
	Email string
}

// ConcluirPreCadastroUseCase cria a conta direto a partir de um token de
// pré-cadastro, sem uma segunda confirmação por email: quem chega até aqui já
// provou posse do email duas vezes (recebeu o token de cancelamento no email
// original, e o token de pré-cadastro embutido nele) — pedir uma terceira
// prova só adiciona fricção. Reaproveita a mesma materialização de conta do
// fluxo de confirmação por email (cria nova ou converte um convidado
// existente, preservando o histórico de agendamentos).
type ConcluirPreCadastroUseCase struct {
	clients     repositorioClient
	providers   buscadorProvider
	preCadastro repositorioPreCadastro
	hasher      hasherSenha
}

// NovoConcluirPreCadastroUseCase cria uma instância de ConcluirPreCadastroUseCase com as dependências injetadas.
func NovoConcluirPreCadastroUseCase(clients repositorioClient, providers buscadorProvider, preCadastro repositorioPreCadastro, hasher hasherSenha) *ConcluirPreCadastroUseCase {
	return &ConcluirPreCadastroUseCase{clients: clients, providers: providers, preCadastro: preCadastro, hasher: hasher}
}

// Executar consome o token de pré-cadastro (uso único) e cria a conta com a
// senha informada. Retorna ErrPreCadastroInvalido para token inexistente, e
// ErrCadastroInvalido se o email já virou conta (prestador ou cliente) no
// meio-tempo.
func (uc *ConcluirPreCadastroUseCase) Executar(in ConcluirPreCadastroInput) (*ConcluirPreCadastroOutput, error) {
	p, err := uc.preCadastro.Consumir(token.Hash(in.Token))
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPreCadastroInvalido
	}

	// o email não pode ter virado prestador nesse meio-tempo — é único no sistema
	prestador, err := uc.providers.BuscarPorEmail(p.Email)
	if err != nil {
		return nil, err
	}
	if prestador != nil {
		return nil, ErrCadastroInvalido
	}

	senhaHash, err := uc.hasher.Gerar(in.Senha)
	if err != nil {
		return nil, err
	}

	c, err := materializarConta(uc.clients, p.Nome, p.Email, p.Telefone, senhaHash)
	if err != nil {
		return nil, err
	}

	return &ConcluirPreCadastroOutput{ID: c.ID, Nome: c.Nome, Email: c.Email}, nil
}
