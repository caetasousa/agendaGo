package dto

import "github.com/go-playground/validator/v10"

var validate = validator.New()

type CadastrarProviderRequest struct {
	Nome     string `json:"nome"     validate:"required,min=2,max=100"`
	Email    string `json:"email"    validate:"required,email"`
	Telefone string `json:"telefone" validate:"required,min=8,max=30"`
	Senha    string `json:"senha"    validate:"required,min=8"`
}

func (r CadastrarProviderRequest) Validar() error {
	return validate.Struct(r)
}

type CadastrarProviderResponse struct {
	ID    string `json:"id"`
	Nome  string `json:"nome"`
	Email string `json:"email"`
}

type AtualizarPreferenciasRequest struct {
	Telefone                     string     `json:"telefone" validate:"required,min=8,max=30"`
	AceitaAgendamentos           bool       `json:"aceitaAgendamentos"`
	DescansoMinutos              int        `json:"descansoMinutos" validate:"min=0"`
	DuracaoAtendimentoMinutos    int        `json:"duracaoAtendimentoMinutos" validate:"min=15,max=1440"`
	HorariosPadrao               []BlocoDTO `json:"horariosPadrao"  validate:"dive"`
	PermiteMarcacaoPeloPrestador bool       `json:"permiteMarcacaoPeloPrestador"`
}

func (r AtualizarPreferenciasRequest) Validar() error {
	return validate.Struct(r)
}

type AtualizarPreferenciasResponse struct {
	Telefone                     string     `json:"telefone"`
	AceitaAgendamentos           bool       `json:"aceitaAgendamentos"`
	DescansoMinutos              int        `json:"descansoMinutos"`
	DuracaoAtendimentoMinutos    int        `json:"duracaoAtendimentoMinutos"`
	HorariosPadrao               []BlocoDTO `json:"horariosPadrao"`
	PermiteMarcacaoPeloPrestador bool       `json:"permiteMarcacaoPeloPrestador"`
}
