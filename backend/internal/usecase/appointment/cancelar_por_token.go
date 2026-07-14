package appointment

import (
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/pkg/token"
)

// DetalheCancelamento descreve o agendamento apontado por um token de
// cancelamento, para a página de confirmação do convidado.
type DetalheCancelamento struct {
	NomePrestador string
	Data          time.Time
	InicioMinutos int
	FimMinutos    int
	Status        appointment.Status
	// PodeCancelar informa se o agendamento ainda pode ser cancelado agora
	// (status cancelável e antecedência mínima respeitada).
	PodeCancelar bool
}

// CancelarPorTokenUseCase permite ao convidado (sem conta) consultar e cancelar
// um agendamento pelo token recebido no email, sem sessão. A autorização de
// sessão é substituída pelo token; a autorização do que pode ser feito
// (antecedência, status) continua no domínio.
type CancelarPorTokenUseCase struct {
	repo         repositorioAppointment
	cancelamento repositorioCancelamento
	providerRepo repositorioProvider
	clientRepo   repositorioClient
	notificador  notificadorAgendamento
	antecedencia time.Duration
	fuso         *time.Location
}

// NovoCancelarPorTokenUseCase cria uma instância de CancelarPorTokenUseCase com as dependências injetadas.
func NovoCancelarPorTokenUseCase(
	repo repositorioAppointment,
	cancelamento repositorioCancelamento,
	providerRepo repositorioProvider,
	clientRepo repositorioClient,
	notificador notificadorAgendamento,
	antecedencia time.Duration,
	fuso *time.Location,
) *CancelarPorTokenUseCase {
	return &CancelarPorTokenUseCase{
		repo:         repo,
		cancelamento: cancelamento,
		providerRepo: providerRepo,
		clientRepo:   clientRepo,
		notificador:  notificador,
		antecedencia: antecedencia,
		fuso:         fuso,
	}
}

// Detalhar resolve o agendamento apontado pelo token para a página de
// confirmação. Retorna ErrTokenCancelamentoInvalido se o token ou o
// agendamento não existir.
func (uc *CancelarPorTokenUseCase) Detalhar(tokenPuro string, agora time.Time) (*DetalheCancelamento, error) {
	a, err := uc.buscarAgendamento(token.Hash(tokenPuro), agora)
	if err != nil {
		return nil, err
	}

	p, err := uc.providerRepo.BuscarPorID(a.ProviderID)
	if err != nil {
		return nil, err
	}
	nomePrestador := ""
	if p != nil {
		nomePrestador = p.Nome
	}

	return &DetalheCancelamento{
		NomePrestador: nomePrestador,
		Data:          a.Data,
		InicioMinutos: a.InicioMinutos,
		FimMinutos:    a.FimMinutos,
		Status:        a.Status,
		PodeCancelar:  a.Cancelavel(agora, uc.antecedencia, uc.fuso),
	}, nil
}

// Executar cancela o agendamento apontado pelo token. Passa pelo mesmo método
// de domínio Cancelar, então a regra de antecedência mínima e os status
// canceláveis são respeitados — o token não burla a regra de negócio, só a
// sessão. Consome o token ao cancelar com sucesso (uso único: o mesmo link
// não cancela duas vezes) e notifica o prestador do cancelamento.
func (uc *CancelarPorTokenUseCase) Executar(tokenPuro string, agora time.Time) error {
	tokenHash := token.Hash(tokenPuro)
	a, err := uc.buscarAgendamento(tokenHash, agora)
	if err != nil {
		return err
	}

	if a.ExpirarSeVencido(agora) {
		if err := uc.repo.Atualizar(a); err != nil {
			return err
		}
		return appointment.ErrSolicitacaoExpirada
	}

	if err := a.Cancelar(agora, uc.antecedencia, uc.fuso); err != nil {
		return err
	}
	if err := uc.repo.Atualizar(a); err != nil {
		return err
	}

	uc.cancelamento.Remover(tokenHash)
	uc.notificarPrestador(a)
	return nil
}

// buscarAgendamento resolve o agendamento a partir do hash do token, tratando
// token inexistente/expirado e agendamento inexistente como o mesmo erro
// genérico.
func (uc *CancelarPorTokenUseCase) buscarAgendamento(tokenHash string, agora time.Time) (*appointment.Appointment, error) {
	t, err := uc.cancelamento.BuscarPorTokenHash(tokenHash)
	if err != nil {
		return nil, err
	}
	if t == nil || t.Expirado(agora) {
		return nil, ErrTokenCancelamentoInvalido
	}

	a, err := uc.repo.BuscarPorID(t.AppointmentID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrTokenCancelamentoInvalido
	}
	return a, nil
}

func (uc *CancelarPorTokenUseCase) notificarPrestador(a *appointment.Appointment) {
	p, err := uc.providerRepo.BuscarPorID(a.ProviderID)
	if err != nil || p == nil {
		return
	}
	c, err := uc.clientRepo.BuscarPorID(a.ClientID)
	if err != nil || c == nil {
		return
	}

	uc.notificador.NotificarCancelamento(NotificacaoAgendamento{
		NomePrestador:         p.Nome,
		EmailPrestador:        p.Email,
		NomeCliente:           c.Nome,
		EmailCliente:          c.Email,
		Data:                  a.Data,
		InicioMinutos:         a.InicioMinutos,
		FimMinutos:            a.FimMinutos,
		CanceladoPorPrestador: false,
	})
}
