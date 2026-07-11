package dto

import "github.com/go-playground/validator/v10"

var validate = validator.New()

type CadastrarProviderRequest struct {
	Nome  string `json:"nome"  validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email"`
	Senha string `json:"senha" validate:"required,min=8"`
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
	AceitaAgendamentos bool       `json:"aceitaAgendamentos"`
	DescansoMinutos    int        `json:"descansoMinutos" validate:"min=0"`
	HorariosPadrao     []BlocoDTO `json:"horariosPadrao"  validate:"dive"`
}

func (r AtualizarPreferenciasRequest) Validar() error {
	return validate.Struct(r)
}

type AtualizarPreferenciasResponse struct {
	AceitaAgendamentos bool       `json:"aceitaAgendamentos"`
	DescansoMinutos    int        `json:"descansoMinutos"`
	HorariosPadrao     []BlocoDTO `json:"horariosPadrao"`
}
