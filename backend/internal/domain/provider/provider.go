package provider

import (
	"errors"
	"time"

	"agendago/internal/domain/availability"
)

// Provider representa a AGENDA de um prestador de serviço: expediente,
// duração, buffer e se está ofertando horários.
//
// Identidade — email, senha, telefone, situação da conta — não mora aqui: é do
// domínio usuario, ligado a esta agenda pelo domínio membro. A separação
// existe para que uma segunda pessoa (recepção, sócia) possa operar a mesma
// agenda com login próprio.
type Provider struct {
	ID                        string
	Nome                      string
	AceitaAgendamentos        bool
	DescansoMinutos           int
	DuracaoAtendimentoMinutos int
	HorariosPadrao            []availability.TimeBlock
	// PermiteMarcacaoPeloPrestador controla se o próprio prestador pode
	// registrar agendamentos na própria agenda (cliente que ligou, por
	// exemplo). Nasce true — é uma capacidade que já existe por padrão,
	// desativável em Preferências.
	PermiteMarcacaoPeloPrestador bool
	CriadoEm                     time.Time
	AtualizadoEm                 time.Time
}

var (
	// ErrNomeObrigatorio é retornado quando o nome da agenda está vazio.
	ErrNomeObrigatorio = errors.New("nome é obrigatório")
	// ErrDescansoInvalido é retornado quando o tempo de descanso é negativo.
	ErrDescansoInvalido = errors.New("descanso não pode ser negativo")
	// ErrDuracaoInvalida é retornado quando a duração do atendimento está fora de [15, 1440] minutos.
	ErrDuracaoInvalida = errors.New("duração do atendimento deve ficar entre 15 minutos e um dia")
)

// Novo cria uma agenda com atendimentos desativados por padrão. Retorna erro
// se o nome estiver vazio. Credencial nenhuma passa por aqui: quem loga é o
// usuario, ligado a esta agenda por um membro.
func Novo(id, nome string) (*Provider, error) {
	if nome == "" {
		return nil, ErrNomeObrigatorio
	}

	agora := time.Now()
	return &Provider{
		ID:                           id,
		Nome:                         nome,
		AceitaAgendamentos:           false,
		DescansoMinutos:              0,
		DuracaoAtendimentoMinutos:    duracaoAtendimentoSugerida,
		HorariosPadrao:               horariosComerciaisPadrao,
		PermiteMarcacaoPeloPrestador: false,
		CriadoEm:                     agora,
		AtualizadoEm:                 agora,
	}, nil
}

// duracaoAtendimentoSugerida é a duração inicial de um atendimento (1h) para
// um prestador recém-criado — editável a qualquer momento em Preferências.
// Enquanto não existe o domínio de serviços, a duração é única por prestador.
const duracaoAtendimentoSugerida = 60

// horariosComerciaisPadrao é o expediente sugerido a um prestador recém-criado
// — 08:00–12:00 e 14:00–18:00 — editável a qualquer momento em Preferências.
var horariosComerciaisPadrao = []availability.TimeBlock{
	{InicioMinutos: 8 * 60, FimMinutos: 12 * 60},
	{InicioMinutos: 14 * 60, FimMinutos: 18 * 60},
}

// DefinirHorariosPadrao substitui o expediente padrão do prestador (usado em
// dias úteis sem definição própria). Aceita lista vazia (nenhum horário
// padrão) e normaliza os blocos (ordena e mescla adjacentes) com as mesmas
// regras de TimeBlock.
func (p *Provider) DefinirHorariosPadrao(blocos []availability.TimeBlock) error {
	normalizados, err := availability.NormalizarBlocos(blocos)
	if err != nil {
		return err
	}
	p.HorariosPadrao = normalizados
	p.AtualizadoEm = time.Now()
	return nil
}

// AtivarAgenda habilita o prestador a receber agendamentos.
func (p *Provider) AtivarAgenda() {
	p.AceitaAgendamentos = true
	p.AtualizadoEm = time.Now()
}

// DesativarAgenda impede o prestador de receber novos agendamentos.
func (p *Provider) DesativarAgenda() {
	p.AceitaAgendamentos = false
	p.AtualizadoEm = time.Now()
}

// AtivarMarcacaoPeloPrestador permite que o próprio prestador registre
// agendamentos na própria agenda.
func (p *Provider) AtivarMarcacaoPeloPrestador() {
	p.PermiteMarcacaoPeloPrestador = true
	p.AtualizadoEm = time.Now()
}

// DesativarMarcacaoPeloPrestador impede o prestador de registrar
// agendamentos na própria agenda — só sobra o fluxo normal de solicitação.
func (p *Provider) DesativarMarcacaoPeloPrestador() {
	p.PermiteMarcacaoPeloPrestador = false
	p.AtualizadoEm = time.Now()
}

// DefinirDescanso define o intervalo em minutos entre atendimentos.
// Retorna erro se o valor for negativo.
func (p *Provider) DefinirDescanso(minutos int) error {
	if minutos < 0 {
		return ErrDescansoInvalido
	}
	p.DescansoMinutos = minutos
	p.AtualizadoEm = time.Now()
	return nil
}

// DefinirDuracaoAtendimento define a duração em minutos de um atendimento —
// o tamanho de cada slot ofertado. Retorna erro fora de [15, 1440].
func (p *Provider) DefinirDuracaoAtendimento(minutos int) error {
	if minutos < 15 || minutos > 24*60 {
		return ErrDuracaoInvalida
	}
	p.DuracaoAtendimentoMinutos = minutos
	p.AtualizadoEm = time.Now()
	return nil
}
