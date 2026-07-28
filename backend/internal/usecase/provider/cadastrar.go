package provider

import (
	"errors"
	"time"

	"agendago/internal/domain/membro"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/session"
	"agendago/internal/domain/signup"
	"agendago/internal/domain/usuario"
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
	usuarios  repositorioUsuarioCadastro
	clients   buscadorClient
	admins    buscadorAdmin
	pendentes repositorioCadastroPendente
	enviador  enviadorCadastro
	hasher    hasherSenha
}

// NovoSolicitarCadastroUseCase cria uma instância de SolicitarCadastroUseCase com as dependências injetadas.
func NovoSolicitarCadastroUseCase(
	usuarios repositorioUsuarioCadastro,
	clients buscadorClient,
	admins buscadorAdmin,
	pendentes repositorioCadastroPendente,
	enviador enviadorCadastro,
	hasher hasherSenha,
) *SolicitarCadastroUseCase {
	return &SolicitarCadastroUseCase{usuarios: usuarios, clients: clients, admins: admins, pendentes: pendentes, enviador: enviador, hasher: hasher}
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

	// o email do admin é reservado: não vira conta de prestador. Retorna em
	// silêncio (sem pendente e sem email), como no cadastro de cliente — a
	// resposta e o timing são idênticos, sem vazar que o email é o do admin.
	admin, err := uc.admins.BuscarPorEmail(in.Email)
	if err != nil {
		return err
	}
	if admin != nil {
		return nil
	}

	prestador, err := uc.usuarios.BuscarPorEmail(in.Email)
	if err != nil {
		return err
	}
	if prestador != nil {
		// Saúda com o nome digitado agora, não com o da conta que já existe: a
		// conta é de quem recebe o email, e o nome dela não precisa voltar
		// para quem tentou o cadastro.
		uc.enviador.EnviarAvisoContaExistente(in.Email, in.Nome)
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
	usuarios  repositorioUsuarioCadastro
	membros   repositorioMembroCadastro
	clients   buscadorClient
	admins    buscadorAdmin
	pendentes repositorioCadastroPendente
}

// NovoConfirmarCadastroUseCase cria uma instância de ConfirmarCadastroUseCase com as dependências injetadas.
func NovoConfirmarCadastroUseCase(
	repo repositorioCadastrar,
	usuarios repositorioUsuarioCadastro,
	membros repositorioMembroCadastro,
	clients buscadorClient,
	admins buscadorAdmin,
	pendentes repositorioCadastroPendente,
) *ConfirmarCadastroUseCase {
	return &ConfirmarCadastroUseCase{repo: repo, usuarios: usuarios, membros: membros, clients: clients, admins: admins, pendentes: pendentes}
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

	// o email do admin é reservado (ver buscadorAdmin): mesmo com o token
	// válido, não materializa conta de prestador com ele
	admin, err := uc.admins.BuscarPorEmail(pendente.Email)
	if err != nil {
		return nil, err
	}
	if admin != nil {
		return nil, ErrCadastroInvalido
	}

	prestador, err := uc.usuarios.BuscarPorEmail(pendente.Email)
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

	// Três peças, nesta ordem: a conta de quem loga, a agenda que ela opera, e
	// o vínculo de dono entre as duas. Sem transação, como o resto dos
	// usecases — uma falha no meio deixa conta sem vínculo, que o login trata
	// como credencial inválida em vez de deixar entrar pela metade.
	u, err := usuario.Novo(uuid.NewString(), pendente.Email, pendente.Telefone, pendente.SenhaHash)
	if err != nil {
		return nil, err
	}
	if err := uc.usuarios.Salvar(u); err != nil {
		return nil, err
	}

	p, err := provider.Novo(uuid.NewString(), pendente.Nome)
	if err != nil {
		return nil, err
	}
	if err := uc.repo.Salvar(p); err != nil {
		return nil, err
	}

	vinculo, err := membro.Novo(uuid.NewString(), u.ID, p.ID, membro.PapelDono)
	if err != nil {
		return nil, err
	}
	if err := uc.membros.Salvar(vinculo); err != nil {
		return nil, err
	}

	uc.pendentes.RemoverExpirados()
	return &ConfirmarCadastroOutput{ID: p.ID, Nome: p.Nome, Email: u.Email}, nil
}
