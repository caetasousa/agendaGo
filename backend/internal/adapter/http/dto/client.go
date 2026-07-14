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

// PreCadastroResponse contém os dados do convidado para pré-preencher o
// formulário de cadastro.
type PreCadastroResponse struct {
	Nome     string `json:"nome"`
	Email    string `json:"email"`
	Telefone string `json:"telefone"`
}

// ConcluirPreCadastroRequest contém a senha escolhida para a conta a criar a
// partir do token de pré-cadastro (na URL, não no corpo).
type ConcluirPreCadastroRequest struct {
	Senha string `json:"senha" validate:"required,min=8"`
}

func (r ConcluirPreCadastroRequest) Validar() error {
	return validate.Struct(r)
}

// ConcluirPreCadastroResponse contém os dados da conta recém-criada, para o
// frontend logar em seguida com o email + senha que a pessoa acabou de digitar.
type ConcluirPreCadastroResponse struct {
	Email string `json:"email"`
	Nome  string `json:"nome"`
}
