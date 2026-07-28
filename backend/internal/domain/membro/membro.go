// Vínculo entre quem loga (usuario) e a agenda que essa pessoa opera
// (provider). É aqui que mora a resposta para "esta pessoa pode mexer nesta
// agenda, e até onde".
package membro

import (
	"errors"
	"time"
)

// Papel identifica o que uma pessoa pode fazer na agenda a que está vinculada.
type Papel string

const (
	// PapelDono identifica quem é responsável pela agenda: além de operá-la,
	// administra a própria conta e quem mais tem acesso.
	PapelDono Papel = "dono"
	// PapelOperador identifica quem opera a agenda no dia a dia (recepção,
	// secretária) sem poder sobre a conta nem sobre os outros vínculos.
	PapelOperador Papel = "operador"
)

var (
	// ErrPapelInvalido é retornado quando o papel não é "dono" nem "operador".
	ErrPapelInvalido = errors.New("papel inválido")
	// ErrUsuarioObrigatorio é retornado quando o vínculo não aponta para um usuário.
	ErrUsuarioObrigatorio = errors.New("usuário é obrigatório")
	// ErrProviderObrigatorio é retornado quando o vínculo não aponta para uma agenda.
	ErrProviderObrigatorio = errors.New("prestador é obrigatório")
)

// Membro representa o vínculo de um usuário com a agenda de um prestador.
type Membro struct {
	ID         string
	UsuarioID  string
	ProviderID string
	Papel      Papel
	CriadoEm   time.Time
}

// Novo cria um vínculo entre um usuário e uma agenda. Retorna erro se algum
// dos lados estiver vazio ou se o papel não for reconhecido.
func Novo(id, usuarioID, providerID string, papel Papel) (*Membro, error) {
	if usuarioID == "" {
		return nil, ErrUsuarioObrigatorio
	}
	if providerID == "" {
		return nil, ErrProviderObrigatorio
	}
	if !papel.Valido() {
		return nil, ErrPapelInvalido
	}

	return &Membro{
		ID:         id,
		UsuarioID:  usuarioID,
		ProviderID: providerID,
		Papel:      papel,
		CriadoEm:   time.Now(),
	}, nil
}

// Valido informa se o papel é um dos reconhecidos pelo domínio. O banco guarda
// papel como VARCHAR sem CHECK justamente porque quem decide isso é aqui.
func (p Papel) Valido() bool {
	return p == PapelDono || p == PapelOperador
}

// PodeGerenciarAgenda informa se o membro pode mexer no expediente, nas
// preferências e nos agendamentos da agenda. Hoje os dois papéis podem — o
// método existe porque a pergunta é do domínio, e o dia em que um papel só
// puder ler, a resposta muda aqui e em nenhum outro lugar.
func (m *Membro) PodeGerenciarAgenda() bool {
	return m.Papel == PapelDono || m.Papel == PapelOperador
}

// PodeAdministrarConta informa se o membro pode mexer na própria conta
// (trocar senha, encerrar) e em quem mais tem acesso à agenda. Só o dono pode:
// é o que separa a pessoa que responde pela agenda de quem só a opera.
func (m *Membro) PodeAdministrarConta() bool {
	return m.Papel == PapelDono
}
