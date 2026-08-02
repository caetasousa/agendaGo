// Package lgpd atende os dois direitos que exigem código: portabilidade
// (exportar o que se tem sobre a pessoa) e exclusão (parar de identificá-la).
//
// A exclusão é ANONIMIZAÇÃO, não DELETE, e a razão é estrutural:
// appointments.client_id tem ON DELETE CASCADE, então apagar a linha do
// cliente destruiria o histórico do prestador junto. Esse histórico é dado de
// outra pessoa, que ela tem obrigação profissional e fiscal de manter. O
// direito do cliente alcança a identificação dele, não o registro de que o
// atendimento existiu.
package lgpd

import (
	"errors"
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/domain/auditoria"
	"agendago/internal/domain/client"
	"agendago/internal/pkg/paging"
)

var (
	// ErrClienteNaoEncontrado é retornado quando a conta não existe.
	ErrClienteNaoEncontrado = errors.New("cliente não encontrado")
	// ErrJaAnonimizado é retornado ao pedir exclusão de quem já foi anonimizado.
	ErrJaAnonimizado = errors.New("esta conta já foi removida")
)

// maxAgendamentosExportados limita a exportação. Reusa a paginação existente em
// vez de carregar o histórico inteiro em memória — um cliente antigo pode ter
// centenas de agendamentos, e a rota responde num JSON só.
const maxAgendamentosExportados = 500

type repositorioClient interface {
	BuscarPorID(id string) (*client.Client, error)
	Atualizar(c *client.Client) error
}

type repositorioAppointment interface {
	ListarPorCliente(clientID string, pag paging.Pagina) ([]*appointment.Appointment, int, error)
}

// removedorDeVinculos apaga o que dá acesso à conta. São agregados diferentes,
// cada um com seu repositório; o usecase orquestra.
type repositorioSessao interface {
	RemoverDoUsuario(userID string) error
}

type repositorioResetSenha interface {
	RemoverDoUsuario(userID string) error
}

type repositorioIdentidadeSocial interface {
	RemoverDoUsuario(userID string) error
}

// registradorAuditoria grava a trilha. Erro ao registrar NÃO derruba a
// operação: perder uma linha de trilha é ruim, deixar de atender um pedido de
// exclusão por causa dela é pior.
type registradorAuditoria interface {
	Registrar(reg *auditoria.Registro) error
}

// AgendamentoExportado é um agendamento na visão de quem pediu os próprios
// dados. Não traz o id do prestador nem observação interna — o que interessa
// é quando foi e em que situação terminou.
type AgendamentoExportado struct {
	Data          time.Time
	InicioMinutos int
	FimMinutos    int
	Status        string
	CriadoEm      time.Time
}

// DadosExportados é o pacote de portabilidade.
type DadosExportados struct {
	ID             string
	Nome           string
	Email          string
	Telefone       string
	CriadoEm       time.Time
	Agendamentos   []AgendamentoExportado
	TotalNoPeriodo int
	// Truncado avisa que o histórico passou do teto e nem tudo veio — sem isso
	// a pessoa acharia que recebeu o conjunto completo.
	Truncado bool
}
