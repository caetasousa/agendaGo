// DTOs da equipe de uma agenda: convidar alguém, aceitar o convite e listar
// quem tem acesso.
package dto

import "time"

// ConvidarMembroRequest é o corpo do convite. O papel é opcional e assume
// "operador" quando vazio — hoje é o único papel convidável, já que o dono é
// definido no cadastro da agenda.
type ConvidarMembroRequest struct {
	Email string `json:"email" validate:"required,email,max=255"`
	Papel string `json:"papel" validate:"omitempty,oneof=operador"`
}

// Validar aplica as regras de formato do corpo da requisição.
func (r ConvidarMembroRequest) Validar() error {
	return validate.Struct(r)
}

// AceitarConviteRequest traz o que a pessoa convidada preenche. O email não
// vem daqui: é o do convite, para ninguém redirecionar um convite alheio.
type AceitarConviteRequest struct {
	Token    string `json:"token" validate:"required"`
	Telefone string `json:"telefone" validate:"required,min=8,max=30"`
	Senha    string `json:"senha" validate:"required,min=8,max=72"`
}

// Validar aplica as regras de formato do corpo da requisição.
func (r AceitarConviteRequest) Validar() error {
	return validate.Struct(r)
}

// ConviteResponse descreve o convite na tela de aceite, antes de a pessoa
// decidir.
type ConviteResponse struct {
	Email      string `json:"email"`
	NomeAgenda string `json:"nomeAgenda"`
	Papel      string `json:"papel"`
}

// MembroResponse descreve quem tem acesso à agenda.
type MembroResponse struct {
	ID       string    `json:"id"`
	Email    string    `json:"email"`
	Papel    string    `json:"papel"`
	Ativo    bool      `json:"ativo"`
	EhDono   bool      `json:"ehDono"`
	CriadoEm time.Time `json:"criadoEm"`
}

// ConvitePendenteResponse descreve um convite emitido e ainda não aceito.
type ConvitePendenteResponse struct {
	Email    string    `json:"email"`
	Papel    string    `json:"papel"`
	ExpiraEm time.Time `json:"expiraEm"`
}

// EquipeResponse reúne quem já entrou e quem ainda foi só convidado.
type EquipeResponse struct {
	Membros   []MembroResponse          `json:"membros"`
	Pendentes []ConvitePendenteResponse `json:"pendentes"`
}
