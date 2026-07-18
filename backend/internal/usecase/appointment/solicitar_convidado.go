package appointment

import (
	"time"

	"agendago/internal/domain/client"

	"github.com/google/uuid"
)

// SolicitarConvidadoInput contém os dados do agendamento feito sem cadastro:
// além do slot, o nome/email/telefone de contato do convidado e uma
// observação livre e opcional, visível ao prestador na lista de agendamentos.
type SolicitarConvidadoInput struct {
	ProviderID    string
	Data          time.Time
	InicioMinutos int
	Nome          string
	Email         string
	Telefone      string
	Agora         time.Time
	Observacao    string
}

// SolicitarConvidadoUseCase cria (ou reusa) um cliente convidado a partir dos
// dados informados e solicita o agendamento. Reaproveita a barreira
// anti-overbooking de SolicitarUseCase. Como o convidado não tem conta, ele
// recebe na hora um email com o link de cancelamento por token e o link
// direto para criar a conta (pré-preenchido).
type SolicitarConvidadoUseCase struct {
	solicitar     *SolicitarUseCase
	clientRepo    repositorioClient
	providerRepo  repositorioProvider
	cancelamentos repositorioCancelamento
	preCadastros  repositorioPreCadastro
	notificador   notificadorAgendamento
}

// NovoSolicitarConvidadoUseCase cria uma instância de SolicitarConvidadoUseCase com as dependências injetadas.
func NovoSolicitarConvidadoUseCase(
	solicitar *SolicitarUseCase,
	clientRepo repositorioClient,
	providerRepo repositorioProvider,
	cancelamentos repositorioCancelamento,
	preCadastros repositorioPreCadastro,
	notificador notificadorAgendamento,
) *SolicitarConvidadoUseCase {
	return &SolicitarConvidadoUseCase{
		solicitar:     solicitar,
		clientRepo:    clientRepo,
		providerRepo:  providerRepo,
		cancelamentos: cancelamentos,
		preCadastros:  preCadastros,
		notificador:   notificador,
	}
}

// Executar resolve o cliente do agendamento e reserva o slot. Se já existe um
// convidado com o email informado, reusa esse convidado — banido não pode
// agendar. E-mail de conta registrada é rejeitado: sem verificação de posse,
// aceitar permitiria criar agendamentos dentro da conta de um terceiro.
// Caso o e-mail seja inédito, cria um convidado novo. Com a reserva feita,
// envia ao convidado o email com o link de cancelamento e o convite de conta.
func (uc *SolicitarConvidadoUseCase) Executar(in SolicitarConvidadoInput) (*SolicitarOutput, error) {
	existente, err := uc.clientRepo.BuscarPorEmail(in.Email)
	if err != nil {
		return nil, err
	}

	var convidado *client.Client
	if existente != nil {
		if existente.TemConta() {
			return nil, ErrEmailTemConta
		}
		if !existente.Ativo {
			return nil, ErrClientInativo
		}
		convidado = existente
	} else {
		novo, err := client.NovoConvidado(uuid.NewString(), in.Nome, in.Email, in.Telefone)
		if err != nil {
			return nil, err
		}
		if err := uc.clientRepo.Salvar(novo); err != nil {
			return nil, err
		}
		convidado = novo
	}

	out, err := uc.solicitar.reservar(in.ProviderID, convidado.ID, in.Data, in.InicioMinutos, in.Agora, in.Observacao)
	if err != nil {
		return nil, err
	}

	uc.notificarConvidado(out, convidado)
	return out, nil
}

// notificarConvidado delega à notificação compartilhada de convidado (tokens
// de cancelamento e pré-cadastro + email de solicitação).
func (uc *SolicitarConvidadoUseCase) notificarConvidado(out *SolicitarOutput, convidado *client.Client) {
	notificarSolicitacaoAoConvidado(uc.providerRepo, uc.cancelamentos, uc.preCadastros, uc.notificador, out, convidado)
}
