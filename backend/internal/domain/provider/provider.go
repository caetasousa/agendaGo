package provider

import (
	"errors"
	"time"

	"agendago/internal/domain/availability"
)

// Provider representa um prestador de serviço no sistema de agendamento.
// Ativo distingue banimento (moderação pelo admin) de AceitaAgendamentos
// (escolha do próprio prestador): um prestador inativo não loga, some da
// vitrine e não oferta horários, mesmo com a agenda ativada.
type Provider struct {
	ID                        string
	Nome                      string
	Email                     string
	Telefone                  string
	SenhaHash                 string
	Ativo                     bool
	AceitaAgendamentos        bool
	DescansoMinutos           int
	DuracaoAtendimentoMinutos int
	HorariosPadrao            []availability.TimeBlock
	CriadoEm                  time.Time
	AtualizadoEm              time.Time
}

var (
	// ErrNomeObrigatorio é retornado quando o nome do prestador está vazio.
	ErrNomeObrigatorio = errors.New("nome é obrigatório")
	// ErrEmailObrigatorio é retornado quando o email do prestador está vazio.
	ErrEmailObrigatorio = errors.New("email é obrigatório")
	// ErrSenhaObrigatoria é retornado quando o hash de senha do prestador está vazio.
	ErrSenhaObrigatoria = errors.New("senha é obrigatória")
	// ErrTelefoneObrigatorio é retornado quando o telefone do prestador está vazio ou é muito curto.
	ErrTelefoneObrigatorio = errors.New("telefone é obrigatório")
	// ErrDescansoInvalido é retornado quando o tempo de descanso é negativo.
	ErrDescansoInvalido = errors.New("descanso não pode ser negativo")
	// ErrDuracaoInvalida é retornado quando a duração do atendimento está fora de [15, 1440] minutos.
	ErrDuracaoInvalida = errors.New("duração do atendimento deve ficar entre 15 minutos e um dia")
)

// Novo cria um Provider com agenda desativada por padrão. Recebe o hash da
// senha já calculado — o domínio não conhece o algoritmo de hash usado.
// Retorna erro se nome, email ou senhaHash estiverem vazios, ou se o telefone
// for inválido (validação leve: ao menos 8 dígitos).
func Novo(id, nome, email, telefone, senhaHash string) (*Provider, error) {
	if nome == "" {
		return nil, ErrNomeObrigatorio
	}
	if email == "" {
		return nil, ErrEmailObrigatorio
	}
	if senhaHash == "" {
		return nil, ErrSenhaObrigatoria
	}
	if !telefoneValido(telefone) {
		return nil, ErrTelefoneObrigatorio
	}

	agora := time.Now()
	return &Provider{
		ID:                        id,
		Nome:                      nome,
		Email:                     email,
		Telefone:                  telefone,
		SenhaHash:                 senhaHash,
		Ativo:                     true,
		AceitaAgendamentos:        false,
		DescansoMinutos:           0,
		DuracaoAtendimentoMinutos: duracaoAtendimentoSugerida,
		HorariosPadrao:            horariosComerciaisPadrao,
		CriadoEm:                  agora,
		AtualizadoEm:              agora,
	}, nil
}

// telefoneValido faz uma validação leve: exige ao menos 8 dígitos, ignorando
// formatação. Não verifica se o número existe.
func telefoneValido(telefone string) bool {
	digitos := 0
	for _, r := range telefone {
		if r >= '0' && r <= '9' {
			digitos++
		}
	}
	return digitos >= 8
}

// DefinirTelefone atualiza o telefone de contato do prestador (Preferências).
// Retorna erro se o telefone for inválido.
func (p *Provider) DefinirTelefone(telefone string) error {
	if !telefoneValido(telefone) {
		return ErrTelefoneObrigatorio
	}
	p.Telefone = telefone
	p.AtualizadoEm = time.Now()
	return nil
}

// Banir desativa o prestador (moderação pelo admin): ele deixa de logar, some
// da vitrine e para de ofertar horários. Reversível por Reativar.
func (p *Provider) Banir() {
	p.Ativo = false
	p.AtualizadoEm = time.Now()
}

// Reativar reverte um banimento, devolvendo o acesso do prestador.
func (p *Provider) Reativar() {
	p.Ativo = true
	p.AtualizadoEm = time.Now()
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
