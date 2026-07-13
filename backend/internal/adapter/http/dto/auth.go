package dto

type LoginRequest struct {
	Email string `json:"email" validate:"required,email"`
	Senha string `json:"senha" validate:"required"`
}

func (r LoginRequest) Validar() error {
	return validate.Struct(r)
}

type LoginResponse struct {
	ID   string `json:"id"`
	Nome string `json:"nome"`
	Tipo string `json:"tipo"`
}

type RecuperarSenhaRequest struct {
	Email string `json:"email" validate:"required,email"`
}

func (r RecuperarSenhaRequest) Validar() error {
	return validate.Struct(r)
}

type RedefinirSenhaRequest struct {
	Token     string `json:"token" validate:"required"`
	NovaSenha string `json:"novaSenha" validate:"required,min=8"`
}

func (r RedefinirSenhaRequest) Validar() error {
	return validate.Struct(r)
}

type MeResponse struct {
	ID                        string     `json:"id"`
	Nome                      string     `json:"nome"`
	Email                     string     `json:"email"`
	Telefone                  string     `json:"telefone,omitempty"`
	Tipo                      string     `json:"tipo"`
	AceitaAgendamentos        *bool      `json:"aceitaAgendamentos,omitempty"`
	DescansoMinutos           *int       `json:"descansoMinutos,omitempty"`
	DuracaoAtendimentoMinutos *int       `json:"duracaoAtendimentoMinutos,omitempty"`
	HorariosPadrao            []BlocoDTO `json:"horariosPadrao,omitempty"`
}
