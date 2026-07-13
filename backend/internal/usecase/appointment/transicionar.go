// Usecases das transições da máquina de estados disparadas pelo prestador
// (confirmar, recusar, realizado, não compareceu) e o cancelamento, que pode
// partir de qualquer um dos dois lados.
package appointment

import (
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/domain/cancellation"
	"agendago/internal/domain/session"
	"agendago/internal/pkg/token"
)

// TransicionarInput identifica o agendamento e quem está agindo sobre ele.
// UsuarioID e Tipo vêm da identidade da sessão autenticada.
type TransicionarInput struct {
	AgendamentoID string
	UsuarioID     string
	Tipo          session.TipoUsuario
	Agora         time.Time
}

// TransicionarUseCase concentra as transições de status de um agendamento
// existente. As regras de cada transição vivem no domínio; aqui ficam a
// autorização (dono do recurso), a expiração lazy e a persistência.
type TransicionarUseCase struct {
	repo             repositorioAppointment
	providerRepo     repositorioProvider
	clientRepo       repositorioClient
	cancelamentoRepo repositorioCancelamento
	notificador      notificadorAgendamento
	antecedencia     time.Duration
	fuso             *time.Location
}

// NovoTransicionarUseCase cria uma instância de TransicionarUseCase com as dependências injetadas.
func NovoTransicionarUseCase(
	repo repositorioAppointment,
	providerRepo repositorioProvider,
	clientRepo repositorioClient,
	cancelamentoRepo repositorioCancelamento,
	notificador notificadorAgendamento,
	antecedencia time.Duration,
	fuso *time.Location,
) *TransicionarUseCase {
	return &TransicionarUseCase{
		repo:             repo,
		providerRepo:     providerRepo,
		clientRepo:       clientRepo,
		cancelamentoRepo: cancelamentoRepo,
		notificador:      notificador,
		antecedencia:     antecedencia,
		fuso:             fuso,
	}
}

// Confirmar aceita uma solicitação pendente do prestador autenticado. Ao
// confirmar um agendamento de convidado, gera um token de cancelamento para
// ele receber no email — sua única via de cancelamento, já que não tem conta.
func (uc *TransicionarUseCase) Confirmar(in TransicionarInput) error {
	a, err := uc.transicionarComoPrestador(in, func(a *appointment.Appointment) error {
		return a.Confirmar(in.Agora)
	})
	if err != nil {
		return err
	}
	uc.notificarConfirmacao(a)
	return nil
}

// Recusar nega uma solicitação pendente do prestador autenticado.
func (uc *TransicionarUseCase) Recusar(in TransicionarInput) error {
	a, err := uc.transicionarComoPrestador(in, func(a *appointment.Appointment) error {
		return a.Recusar(in.Agora)
	})
	if err != nil {
		return err
	}
	uc.notificar(a, uc.notificador.NotificarRecusa, false)
	return nil
}

// MarcarRealizado conclui um agendamento confirmado cujo horário já passou.
func (uc *TransicionarUseCase) MarcarRealizado(in TransicionarInput) error {
	_, err := uc.transicionarComoPrestador(in, func(a *appointment.Appointment) error {
		return a.MarcarRealizado(in.Agora, uc.fuso)
	})
	return err
}

// MarcarNaoCompareceu registra a ausência do cliente num agendamento confirmado.
func (uc *TransicionarUseCase) MarcarNaoCompareceu(in TransicionarInput) error {
	_, err := uc.transicionarComoPrestador(in, func(a *appointment.Appointment) error {
		return a.MarcarNaoCompareceu(in.Agora, uc.fuso)
	})
	return err
}

// Cancelar encerra o agendamento a pedido do cliente ou do prestador. O
// prestador não cancela solicitações pendentes — para isso existe Recusar.
// As regras de antecedência ficam no domínio.
func (uc *TransicionarUseCase) Cancelar(in TransicionarInput) error {
	a, err := uc.carregarDoUsuario(in)
	if err != nil {
		return err
	}

	if a.ExpirarSeVencido(in.Agora) {
		if err := uc.repo.Atualizar(a); err != nil {
			return err
		}
		return appointment.ErrSolicitacaoExpirada
	}

	if in.Tipo == session.TipoProvider && a.Status == appointment.StatusSolicitado {
		return appointment.ErrTransicaoInvalida
	}

	if err := a.Cancelar(in.Agora, uc.antecedencia, uc.fuso); err != nil {
		return err
	}
	if err := uc.repo.Atualizar(a); err != nil {
		return err
	}
	uc.notificar(a, uc.notificador.NotificarCancelamento, in.Tipo == session.TipoProvider)
	return nil
}

// transicionarComoPrestador carrega o agendamento do prestador autenticado,
// efetiva a expiração lazy e aplica a transição do domínio.
func (uc *TransicionarUseCase) transicionarComoPrestador(in TransicionarInput, transicao func(*appointment.Appointment) error) (*appointment.Appointment, error) {
	if in.Tipo != session.TipoProvider {
		return nil, ErrAgendamentoNaoEncontrado
	}

	a, err := uc.carregarDoUsuario(in)
	if err != nil {
		return nil, err
	}

	if a.ExpirarSeVencido(in.Agora) {
		if err := uc.repo.Atualizar(a); err != nil {
			return nil, err
		}
		return nil, appointment.ErrSolicitacaoExpirada
	}

	if err := transicao(a); err != nil {
		return nil, err
	}
	if err := uc.repo.Atualizar(a); err != nil {
		return nil, err
	}
	return a, nil
}

// notificar resolve nome/email das duas partes e dispara o evento
// correspondente. Best-effort: se não conseguir resolver os dados, a
// notificação é silenciosamente pulada.
func (uc *TransicionarUseCase) notificar(a *appointment.Appointment, evento func(NotificacaoAgendamento), canceladoPorPrestador bool) {
	p, err := uc.providerRepo.BuscarPorID(a.ProviderID)
	if err != nil || p == nil {
		return
	}
	c, err := uc.clientRepo.BuscarPorID(a.ClientID)
	if err != nil || c == nil {
		return
	}

	evento(NotificacaoAgendamento{
		NomePrestador:         p.Nome,
		EmailPrestador:        p.Email,
		NomeCliente:           c.Nome,
		EmailCliente:          c.Email,
		Data:                  a.Data,
		InicioMinutos:         a.InicioMinutos,
		FimMinutos:            a.FimMinutos,
		CanceladoPorPrestador: canceladoPorPrestador,
	})
}

// notificarConfirmacao avisa o cliente da confirmação. Para um convidado (sem
// conta), gera e persiste um token de cancelamento e o inclui no email — é a
// única forma de ele cancelar. Best-effort: falha ao gerar/persistir o token
// não bloqueia a confirmação (só sai sem link de cancelamento).
func (uc *TransicionarUseCase) notificarConfirmacao(a *appointment.Appointment) {
	p, err := uc.providerRepo.BuscarPorID(a.ProviderID)
	if err != nil || p == nil {
		return
	}
	c, err := uc.clientRepo.BuscarPorID(a.ClientID)
	if err != nil || c == nil {
		return
	}

	var tokenCancelamento string
	if !c.TemConta() {
		if t, err := token.Gerar(); err == nil {
			if err := uc.cancelamentoRepo.Salvar(cancellation.Novo(token.Hash(t), a.ID)); err == nil {
				tokenCancelamento = t
			}
		}
	}

	uc.notificador.NotificarConfirmacao(NotificacaoAgendamento{
		NomePrestador:     p.Nome,
		EmailPrestador:    p.Email,
		NomeCliente:       c.Nome,
		EmailCliente:      c.Email,
		Data:              a.Data,
		InicioMinutos:     a.InicioMinutos,
		FimMinutos:        a.FimMinutos,
		TokenCancelamento: tokenCancelamento,
	})
}

// carregarDoUsuario busca o agendamento e garante que pertence ao usuário da
// sessão (como prestador ou como cliente, conforme o tipo).
func (uc *TransicionarUseCase) carregarDoUsuario(in TransicionarInput) (*appointment.Appointment, error) {
	a, err := uc.repo.BuscarPorID(in.AgendamentoID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrAgendamentoNaoEncontrado
	}

	dono := (in.Tipo == session.TipoProvider && a.ProviderID == in.UsuarioID) ||
		(in.Tipo == session.TipoClient && a.ClientID == in.UsuarioID)
	if !dono {
		return nil, ErrAgendamentoNaoEncontrado
	}
	return a, nil
}
