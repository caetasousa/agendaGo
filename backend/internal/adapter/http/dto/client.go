package dto

type CadastrarClientRequest struct {
	Nome     string `json:"nome"     validate:"required,min=2,max=100"`
	Email    string `json:"email"    validate:"required,email"`
	Telefone string `json:"telefone" validate:"required,min=8,max=30"`
	Senha    string `json:"senha"    validate:"required,min=8"`
}

func (r CadastrarClientRequest) Validar() error {
	return validate.Struct(r)
}

type ConfirmarCadastroRequest struct {
	Token string `json:"token" validate:"required"`
}

func (r ConfirmarCadastroRequest) Validar() error {
	return validate.Struct(r)
}
