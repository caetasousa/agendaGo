package dto

type CadastrarClientRequest struct {
	Nome  string `json:"nome"  validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email"`
	Senha string `json:"senha" validate:"required,min=8"`
}

func (r CadastrarClientRequest) Validar() error {
	return validate.Struct(r)
}

type CadastrarClientResponse struct {
	ID    string `json:"id"`
	Nome  string `json:"nome"`
	Email string `json:"email"`
}
