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

// ProviderDoMeResponse é a agenda que o usuário autenticado opera, e o papel
// dele nela. Fica num objeto próprio para separar, na resposta, o que é da
// conta (email, telefone) do que é da agenda — quando uma pessoa puder operar
// a agenda de outra, misturar os dois no topo deixaria de fazer sentido.
type ProviderDoMeResponse struct {
	ID                           string     `json:"id"`
	Papel                        string     `json:"papel"`
	AceitaAgendamentos           bool       `json:"aceitaAgendamentos"`
	DescansoMinutos              int        `json:"descansoMinutos"`
	DuracaoAtendimentoMinutos    int        `json:"duracaoAtendimentoMinutos"`
	HorariosPadrao               []BlocoDTO `json:"horariosPadrao"`
	PermiteMarcacaoPeloPrestador bool       `json:"permiteMarcacaoPeloPrestador"`
}

type MeResponse struct {
	ID       string `json:"id"`
	Nome     string `json:"nome"`
	Email    string `json:"email"`
	Telefone string `json:"telefone,omitempty"`
	Tipo     string `json:"tipo"`
	// Provider só vem para o tipo provider — ausente para cliente e admin.
	Provider *ProviderDoMeResponse `json:"provider,omitempty"`
	// TelefonePendente é true quando o prestador entrou via login social e
	// ainda não confirmou um telefone de verdade — o frontend usa isso para
	// travar o painel em Preferências até ele completar o cadastro.
	TelefonePendente bool `json:"telefonePendente,omitempty"`
}
