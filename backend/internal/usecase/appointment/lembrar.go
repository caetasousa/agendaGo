package appointment

import (
	"time"

	"agendago/internal/domain/appointment"
)

// LembrarUseCase envia o lembrete por email dos agendamentos confirmados que
// começam dentro da janela de antecedência configurada e ainda não foram
// lembrados.
type LembrarUseCase struct {
	repo         repositorioAppointment
	providerRepo repositorioProvider
	clientRepo   repositorioClient
	notificador  notificadorAgendamento
	fuso         *time.Location
	antecedencia time.Duration
}

// NovoLembrarUseCase cria uma instância de LembrarUseCase com as dependências injetadas.
func NovoLembrarUseCase(
	repo repositorioAppointment,
	providerRepo repositorioProvider,
	clientRepo repositorioClient,
	notificador notificadorAgendamento,
	fuso *time.Location,
	antecedencia time.Duration,
) *LembrarUseCase {
	return &LembrarUseCase{
		repo:         repo,
		providerRepo: providerRepo,
		clientRepo:   clientRepo,
		notificador:  notificador,
		fuso:         fuso,
		antecedencia: antecedencia,
	}
}

// Executar busca os agendamentos confirmados que começam nas próximas
// antecedencia (tipicamente 24h) e ainda não foram lembrados, reivindica cada
// um via MarcarLembreteEnviado (claim que evita duplicata sob concorrência) e
// dispara a notificação. A janela de datas cobre no máximo dois dias-calendário
// no fuso do sistema; o filtro fino por instante acontece em Go.
func (uc *LembrarUseCase) Executar(agora time.Time) error {
	agoraNoFuso := agora.In(uc.fuso)
	hoje := time.Date(agoraNoFuso.Year(), agoraNoFuso.Month(), agoraNoFuso.Day(), 0, 0, 0, 0, uc.fuso)
	amanha := hoje.AddDate(0, 0, 1)

	candidatos, err := uc.repo.ListarConfirmadosSemLembrete(hoje, amanha)
	if err != nil {
		return err
	}

	for _, a := range candidatos {
		inicio := a.InicioEm(uc.fuso)
		if !agora.Before(inicio) || inicio.Sub(agora) > uc.antecedencia {
			continue
		}

		reivindicado, err := uc.repo.MarcarLembreteEnviado(a.ID, agora)
		if err != nil {
			return err
		}
		if !reivindicado {
			continue
		}

		uc.notificarLembrete(a)
	}

	return nil
}

// notificarLembrete resolve nome/email das duas partes e dispara o evento.
// Best-effort: se não conseguir resolver os dados, a notificação é
// silenciosamente pulada — o lembrete já foi reivindicado, então não há retry.
func (uc *LembrarUseCase) notificarLembrete(a *appointment.Appointment) {
	p, err := uc.providerRepo.BuscarPorID(a.ProviderID)
	if err != nil || p == nil {
		return
	}
	c, err := uc.clientRepo.BuscarPorID(a.ClientID)
	if err != nil || c == nil {
		return
	}

	uc.notificador.NotificarLembrete(NotificacaoAgendamento{
		NomePrestador:  p.Nome,
		EmailPrestador: p.Email,
		NomeCliente:    c.Nome,
		EmailCliente:   c.Email,
		Data:           a.Data,
		InicioMinutos:  a.InicioMinutos,
		FimMinutos:     a.FimMinutos,
	})
}
