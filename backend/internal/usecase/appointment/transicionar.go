// Usecases das transições da máquina de estados disparadas pelo prestador
// (confirmar, recusar, realizado, não compareceu) e o cancelamento, que pode
// partir de qualquer um dos dois lados.
package appointment

import (
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/domain/session"
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
	repo        repositorioAppointment
	antecedencia time.Duration
	fuso        *time.Location
}

// NovoTransicionarUseCase cria uma instância de TransicionarUseCase com as dependências injetadas.
func NovoTransicionarUseCase(repo repositorioAppointment, antecedencia time.Duration, fuso *time.Location) *TransicionarUseCase {
	return &TransicionarUseCase{repo: repo, antecedencia: antecedencia, fuso: fuso}
}

// Confirmar aceita uma solicitação pendente do prestador autenticado.
func (uc *TransicionarUseCase) Confirmar(in TransicionarInput) error {
	return uc.transicionarComoPrestador(in, func(a *appointment.Appointment) error {
		return a.Confirmar(in.Agora)
	})
}

// Recusar nega uma solicitação pendente do prestador autenticado.
func (uc *TransicionarUseCase) Recusar(in TransicionarInput) error {
	return uc.transicionarComoPrestador(in, func(a *appointment.Appointment) error {
		return a.Recusar(in.Agora)
	})
}

// MarcarRealizado conclui um agendamento confirmado cujo horário já passou.
func (uc *TransicionarUseCase) MarcarRealizado(in TransicionarInput) error {
	return uc.transicionarComoPrestador(in, func(a *appointment.Appointment) error {
		return a.MarcarRealizado(in.Agora, uc.fuso)
	})
}

// MarcarNaoCompareceu registra a ausência do cliente num agendamento confirmado.
func (uc *TransicionarUseCase) MarcarNaoCompareceu(in TransicionarInput) error {
	return uc.transicionarComoPrestador(in, func(a *appointment.Appointment) error {
		return a.MarcarNaoCompareceu(in.Agora, uc.fuso)
	})
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
	return uc.repo.Atualizar(a)
}

// transicionarComoPrestador carrega o agendamento do prestador autenticado,
// efetiva a expiração lazy e aplica a transição do domínio.
func (uc *TransicionarUseCase) transicionarComoPrestador(in TransicionarInput, transicao func(*appointment.Appointment) error) error {
	if in.Tipo != session.TipoProvider {
		return ErrAgendamentoNaoEncontrado
	}

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

	if err := transicao(a); err != nil {
		return err
	}
	return uc.repo.Atualizar(a)
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
