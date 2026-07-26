package provider

import (
	"errors"
	"time"

	"agendago/internal/domain/provider"
	"agendago/internal/domain/session"
	"agendago/internal/domain/signup"
	"agendago/internal/pkg/token"

	"github.com/google/uuid"
)

// ErrCadastroInvalido é retornado quando o token de confirmação não existe,
// expirou, não é de um cadastro de prestador, ou o email deixou de estar livre
// nesse meio-tempo — genérico de propósito, para não descrever a um estranho o
// estado do cadastro alheio.
var ErrCadastroInvalido = errors.New("cadastro inválido ou expirado")

// TTLConfirmacaoCadastro é o prazo para confirmar o cadastro a partir da
// solicitação.
const TTLConfirmacaoCadastro = 24 * time.Hour

// SolicitarCadastroInput contém os dados do cadastro a confirmar por email.
type SolicitarCadastroInput struct {
	Nome     string
	Email    string
	Telefone string
	Senha    string
}

// SolicitarCadastroUseCase inicia o cadastro de um prestador: guarda os dados
// pendentes (senha já hasheada) e envia o email de confirmação. A conta não
// nasce aqui — só quando a pessoa prova posse do email.
//
// Antes, o cadastro criava a conta na hora e respondia "email já cadastrado"
// quando o endereço estava em uso. Isso deixava qualquer um descobrir quem tem
// conta no sistema, e permitia publicar na vitrine um prestador com o email de
// outra pessoa. Agora o fluxo é o mesmo do cliente: resposta idêntica em todos
// os casos, e o que muda é só o email que sai.
type SolicitarCadastroUseCase struct {
	repo      repositorioCadastrar
	clients   buscadorClient
	pendentes repositorioCadastroPendente
	enviador  enviadorCadastro
	hasher    hasherSenha
}

// NovoSolicitarCadastroUseCase cria uma instância de SolicitarCadastroUseCase com as dependências injetadas.
func NovoSolicitarCadastroUseCase(
	repo repositorioCadastrar,
	clients buscadorClient,
	pendentes repositorioCadastroPendente,
	enviador enviadorCadastro,
	hasher hasherSenha,
) *SolicitarCadastroUseCase {
	return &SolicitarCadastroUseCase{repo: repo, clients: clients, pendentes: pendentes, enviador: enviador, hasher: hasher}
}

// Executar processa a solicitação. Devolve sempre nil (fora falha real de
// infraestrutura), sem sinalizar de forma nenhuma o que aconteceu:
//
//   - Email já em uso (por prestador, cliente ou convidado): envia o aviso
//     "esse email já está em uso" e não cria pendente.
//   - Email livre: gera o token, guarda o cadastro pendente e envia o link.
//
// A senha é hasheada em todos os caminhos, para equalizar o custo/tempo de
// resposta e nunca guardar texto puro.
func (uc *SolicitarCadastroUseCase) Executar(in SolicitarCadastroInput) error {
	senhaHash, err := uc.hasher.Gerar(in.Senha)
	if err != nil {
		return err
	}

	prestador, err := uc.repo.BuscarPorEmail(in.Email)
	if err != nil {
		return err
	}
	if prestador != nil {
		uc.enviador.EnviarAvisoContaExistente(in.Email, prestador.Nome)
		return nil
	}

	// o email não pode já pertencer a um cliente ou convidado: ele é único no
	// sistema inteiro
	cliente, err := uc.clients.BuscarPorEmail(in.Email)
	if err != nil {
		return err
	}
	if cliente != nil {
		uc.enviador.EnviarAvisoContaExistente(in.Email, cliente.Nome)
		return nil
	}

	// invalida pendentes anteriores do mesmo email: só o último link vale
	if err := uc.pendentes.RemoverPorEmail(in.Email); err != nil {
		return err
	}

	t, err := token.Gerar()
	if err != nil {
		return err
	}
	pendente := signup.Novo(token.Hash(t), in.Nome, in.Email, in.Telefone, senhaHash, session.TipoProvider, TTLConfirmacaoCadastro)
	if err := uc.pendentes.Salvar(pendente); err != nil {
		return err
	}

	uc.enviador.EnviarConfirmacaoCadastroPrestador(in.Email, in.Nome, t, pendente.ExpiraEm)
	return nil
}

// ConfirmarCadastroOutput contém os dados do prestador criado.
type ConfirmarCadastroOutput struct {
	ID    string
	Nome  string
	Email string
}

// ConfirmarCadastroUseCase conclui o cadastro do prestador: consome o token e
// cria a conta.
type ConfirmarCadastroUseCase struct {
	repo      repositorioCadastrar
	clients   buscadorClient
	pendentes repositorioCadastroPendente
}

// NovoConfirmarCadastroUseCase cria uma instância de ConfirmarCadastroUseCase com as dependências injetadas.
func NovoConfirmarCadastroUseCase(repo repositorioCadastrar, clients buscadorClient, pendentes repositorioCadastroPendente) *ConfirmarCadastroUseCase {
	return &ConfirmarCadastroUseCase{repo: repo, clients: clients, pendentes: pendentes}
}

// Executar consome o token (uso único) e cria o prestador. Retorna
// ErrCadastroInvalido para token inexistente, expirado, de outro tipo de conta,
// ou quando o email deixou de estar livre entre o pedido e a confirmação.
//
// A checagem de tipo é o que impede um token de cadastro de cliente de virar
// uma conta de prestador: o tipo vem do registro no banco, nunca da requisição.
func (uc *ConfirmarCadastroUseCase) Executar(tokenPuro string) (*ConfirmarCadastroOutput, error) {
	pendente, err := uc.pendentes.Consumir(token.Hash(tokenPuro))
	if err != nil {
		return nil, err
	}
	if pendente == nil || pendente.Expirado(time.Now()) || pendente.Tipo != session.TipoProvider {
		return nil, ErrCadastroInvalido
	}

	prestador, err := uc.repo.BuscarPorEmail(pendente.Email)
	if err != nil {
		return nil, err
	}
	if prestador != nil {
		return nil, ErrCadastroInvalido
	}
	cliente, err := uc.clients.BuscarPorEmail(pendente.Email)
	if err != nil {
		return nil, err
	}
	if cliente != nil {
		return nil, ErrCadastroInvalido
	}

	p, err := provider.Novo(uuid.NewString(), pendente.Nome, pendente.Email, pendente.Telefone, pendente.SenhaHash)
	if err != nil {
		return nil, err
	}
	if err := uc.repo.Salvar(p); err != nil {
		return nil, err
	}

	uc.pendentes.RemoverExpirados()
	return &ConfirmarCadastroOutput{ID: p.ID, Nome: p.Nome, Email: p.Email}, nil
}
