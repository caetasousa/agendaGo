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

type MeResponse struct {
	ID    string `json:"id"`
	Nome  string `json:"nome"`
	Email string `json:"email"`
	Tipo  string `json:"tipo"`
}
