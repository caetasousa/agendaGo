package membro

import (
	"agendago/internal/domain/convite"
	"agendago/internal/domain/membro"
	"agendago/internal/pkg/token"

	"github.com/google/uuid"
)

// ConvidarInput identifica quem está sendo convidado e para qual agenda.
// ProviderID vem da identidade da sessão, nunca do corpo da requisição.
type ConvidarInput struct {
	ProviderID string
	Email      string
	Papel      membro.Papel
}

// ConvidarUseCase emite o convite para alguém operar uma agenda.
type ConvidarUseCase struct {
	convites  repositorioConvite
	usuarios  repositorioUsuario
	clients   buscadorClient
	admins    buscadorAdmin
	providers buscadorProvider
	enviador  enviadorConvite
}

// NovoConvidarUseCase cria uma instância de ConvidarUseCase com as dependências injetadas.
func NovoConvidarUseCase(
	convites repositorioConvite,
	usuarios repositorioUsuario,
	clients buscadorClient,
	admins buscadorAdmin,
	providers buscadorProvider,
	enviador enviadorConvite,
) *ConvidarUseCase {
	return &ConvidarUseCase{convites: convites, usuarios: usuarios, clients: clients, admins: admins, providers: providers, enviador: enviador}
}

// Executar emite um convite de uso único e dispara o email com o link.
//
// Recusa com ErrEmailIndisponivel quando o email já tem conta, e o motivo é
// concreto nos dois casos:
//
//   - Já é prestador: a pessoa já é dona da própria agenda. Criar um segundo
//     vínculo PARECERIA funcionar e não funcionaria — a resolução da agenda
//     devolve o vínculo mais antigo, então ela continuaria caindo na agenda
//     dela. Aceitar aqui seria entregar um convite silenciosamente inútil.
//   - Já é cliente: o email é único entre clientes e prestadores, e é essa
//     invariante que faz o login unificado funcionar. Duplicar tiraria dela o
//     acesso à própria conta de cliente.
//
// O que destrava o primeiro caso é a escolha de agenda ativa, que ainda não
// existe. Até lá, recusar é a resposta honesta.
//
// Reconvidar o mesmo email substitui o convite anterior, em vez de acumular
// links válidos para o mesmo destino.
func (uc *ConvidarUseCase) Executar(in ConvidarInput) error {
	p, err := uc.providers.BuscarPorID(in.ProviderID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrProviderNaoEncontrado
	}

	if indisponivel, err := uc.emailJaTemConta(in.Email); err != nil {
		return err
	} else if indisponivel {
		return ErrEmailIndisponivel
	}

	t, err := token.Gerar()
	if err != nil {
		return err
	}

	c, err := convite.Novo(token.Hash(t), in.Email, in.ProviderID, in.Papel, TTLConvite)
	if err != nil {
		return err
	}
	if err := uc.convites.Salvar(c); err != nil {
		return err
	}

	uc.enviador.EnviarConviteMembro(in.Email, p.Nome, t, c.ExpiraEm)
	uc.convites.RemoverExpirados()
	return nil
}

// emailJaTemConta cobre os três lugares onde um email pode estar: conta de
// prestador, conta de cliente e o endereço reservado do administrador.
func (uc *ConvidarUseCase) emailJaTemConta(email string) (bool, error) {
	u, err := uc.usuarios.BuscarPorEmail(email)
	if err != nil {
		return false, err
	}
	if u != nil {
		return true, nil
	}

	c, err := uc.clients.BuscarPorEmail(email)
	if err != nil {
		return false, err
	}
	if c != nil {
		return true, nil
	}

	a, err := uc.admins.BuscarPorEmail(email)
	if err != nil {
		return false, err
	}
	return a != nil, nil
}

// CancelarInput identifica o convite pendente a cancelar.
type CancelarInput struct {
	ProviderID string
	Email      string
}

// CancelarConviteUseCase revoga um convite que ainda não foi aceito.
type CancelarConviteUseCase struct {
	convites repositorioConvite
}

// NovoCancelarConviteUseCase cria uma instância de CancelarConviteUseCase.
func NovoCancelarConviteUseCase(convites repositorioConvite) *CancelarConviteUseCase {
	return &CancelarConviteUseCase{convites: convites}
}

// Executar apaga o convite pendente daquele email nesta agenda. Não erra
// quando não há convite: o efeito desejado — aquele link não vale mais — já
// está valendo.
func (uc *CancelarConviteUseCase) Executar(in CancelarInput) error {
	return uc.convites.RemoverPorEmail(in.ProviderID, in.Email)
}

// idNovo isola a geração de identificador, para os testes de usecase não
// dependerem de uuid diretamente.
func idNovo() string { return uuid.NewString() }
