package appointment

import (
	"errors"
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/domain/cancellation"
	"agendago/internal/domain/client"
	"agendago/internal/domain/precadastro"
	"agendago/internal/domain/provider"
)

var (
	// ErrProviderNaoEncontrado é retornado quando o prestador não existe.
	ErrProviderNaoEncontrado = errors.New("prestador não encontrado")
	// ErrClientNaoEncontrado é retornado quando o cliente da sessão não existe mais.
	ErrClientNaoEncontrado = errors.New("cliente não encontrado")
	// ErrClientInativo é retornado quando um cliente banido tenta agendar (por email).
	ErrClientInativo = errors.New("cliente desativado")
	// ErrEmailTemConta é retornado quando o e-mail do convidado pertence a uma
	// conta registrada — sem verificação de posse do e-mail, aceitar criaria
	// agendamentos dentro da conta de um terceiro.
	ErrEmailTemConta = errors.New("este e-mail já tem conta; entre para agendar")
	// ErrAgendamentoNaoEncontrado é retornado quando o agendamento não existe ou não
	// pertence ao usuário — não distingue os casos para não vazar recursos de terceiros.
	ErrAgendamentoNaoEncontrado = errors.New("agendamento não encontrado")
	// ErrHorarioIndisponivel é retornado quando o horário solicitado não é um slot livre.
	ErrHorarioIndisponivel = errors.New("horário indisponível")
	// ErrPeriodoInvalido é retornado quando o período consultado é vazio ou longo demais.
	ErrPeriodoInvalido = errors.New("período inválido")
	// ErrTokenCancelamentoInvalido é retornado quando o token de cancelamento
	// não corresponde a nenhum agendamento — genérico de propósito.
	ErrTokenCancelamentoInvalido = errors.New("link de cancelamento inválido")
	// ErrMarcacaoPeloPrestadorNaoPermitida é retornado quando o prestador
	// desativou, em Preferências, a possibilidade de marcar agendamentos na
	// própria agenda.
	ErrMarcacaoPeloPrestadorNaoPermitida = errors.New("marcação pelo prestador está desativada nas preferências")
)

// TTLPreCadastro é o prazo de validade do token de pré-cadastro entregue ao
// convidado no email — depois disso, o link "Criar minha conta" deixa de
// funcionar e a pessoa precisa se cadastrar pelo caminho normal (com
// confirmação por email).
const TTLPreCadastro = 24 * time.Hour

// TTLCancelamento é o prazo de validade do token de cancelamento entregue ao
// convidado no email. Generoso (cobre o horizonte de agendamento futuro do
// sistema, hoje ~2 meses) porque o link precisa continuar útil até perto do
// horário do atendimento — o token também é consumido (uso único) assim que
// o cancelamento de fato acontece.
const TTLCancelamento = 90 * 24 * time.Hour

// repositorioAppointment persiste e consulta agendamentos. SalvarSeLivre é a
// barreira anti-overbooking: checagem de conflito e escrita atômicas.
type repositorioAppointment interface {
	SalvarSeLivre(a *appointment.Appointment, agora time.Time) error
	BuscarPorID(id string) (*appointment.Appointment, error)
	Atualizar(a *appointment.Appointment) error
	ListarPorPrestador(providerID string) ([]*appointment.Appointment, error)
	ListarPorCliente(clientID string) ([]*appointment.Appointment, error)
	ListarOcupantesPorPeriodo(providerID string, de, ate time.Time, agora time.Time) ([]*appointment.Appointment, error)
	ListarConfirmadosSemLembrete(de, ate time.Time) ([]*appointment.Appointment, error)
	MarcarLembreteEnviado(id string, quando time.Time) (bool, error)
}

// repositorioProvider busca prestadores (duração, buffer e agenda ativa).
type repositorioProvider interface {
	BuscarPorID(id string) (*provider.Provider, error)
}

// repositorioClient busca clientes para validar a sessão e enriquecer
// listagens, e persiste o cliente convidado criado no agendamento sem conta.
type repositorioClient interface {
	BuscarPorID(id string) (*client.Client, error)
	BuscarPorEmail(email string) (*client.Client, error)
	Salvar(c *client.Client) error
}

// repositorioCancelamento persiste e consulta os tokens de cancelamento
// entregues aos convidados. BuscarPorTokenHash não apaga o token na leitura —
// a página de cancelamento lê os detalhes antes de decidir cancelar. Remover
// invalida o token (uso único), chamado só depois que o cancelamento de fato
// acontece.
type repositorioCancelamento interface {
	Salvar(t *cancellation.Token) error
	BuscarPorTokenHash(hash string) (*cancellation.Token, error)
	Remover(hash string) error
	RemoverExpirados() error
}

// repositorioPreCadastro persiste os tokens de pré-cadastro entregues aos
// convidados junto do token de cancelamento — o link "Criar minha conta" leva
// direto à tela de cadastro pré-preenchida.
type repositorioPreCadastro interface {
	Salvar(p *precadastro.PreCadastro) error
}
