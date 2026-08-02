// Compromisso pessoal do prestador — um intervalo do dia que para de ser
// ofertado sem redefinir o expediente.
//
// É o canal genérico de "intervalo ocupado que não é agendamento". O campo
// Origem existe desde o início por isso: hoje só há compromisso criado à mão,
// mas um calendário externo entraria como outra origem, gravando no mesmo
// lugar, sem tocar no cálculo de horários livres.
package ocupacao

import (
	"errors"
	"strings"
	"time"

	"agendago/internal/domain/availability"
)

// Origem identifica de onde veio a ocupação.
type Origem string

const (
	// OrigemManual identifica o compromisso criado pelo próprio prestador.
	OrigemManual Origem = "manual"
)

var (
	// ErrOrigemInvalida é retornado quando a origem não é reconhecida pelo domínio.
	ErrOrigemInvalida = errors.New("origem de ocupação inválida")
	// ErrProviderObrigatorio é retornado quando a ocupação não aponta para uma agenda.
	ErrProviderObrigatorio = errors.New("prestador é obrigatório")
	// ErrTituloLongo é retornado quando o título passa de 120 caracteres.
	ErrTituloLongo = errors.New("título deve ter no máximo 120 caracteres")
	// ErrConflitoComAgendamento é retornado ao tentar criar uma ocupação sobre
	// um horário que já tem agendamento ocupando. O prestador cancela o
	// agendamento primeiro — o sistema não desmarca cliente por conta própria.
	ErrConflitoComAgendamento = errors.New("já existe agendamento neste horário")
)

// tamanhoMaximoTitulo espelha o VARCHAR(120) da coluna.
const tamanhoMaximoTitulo = 120

// Ocupacao é um intervalo de um dia em que o prestador não atende. Título é
// opcional: serve para ele se lembrar do que era, e nunca é exposto a cliente.
type Ocupacao struct {
	ID              string
	ProviderID      string
	Data            time.Time
	InicioMinutos   int
	FimMinutos      int
	Titulo          string
	Origem          Origem
	OrigemExternaID string
	CriadoEm        time.Time
	AtualizadoEm    time.Time
}

// Nova valida e cria um compromisso pessoal. O intervalo passa pelas mesmas
// regras dos blocos de expediente (fim > início, dentro do dia, na
// granularidade), reusando availability.NovoTimeBlock — um compromisso é um
// intervalo do dia como qualquer outro, e ter duas validações divergentes para
// a mesma coisa seria pior do que o acoplamento.
//
// Retorna erro se o prestador estiver vazio, se o título passar de 120
// caracteres, se a origem não for reconhecida, ou o erro do próprio intervalo.
func Nova(id, providerID string, data time.Time, inicioMinutos, fimMinutos int, titulo string, origem Origem) (*Ocupacao, error) {
	if providerID == "" {
		return nil, ErrProviderObrigatorio
	}
	if !origem.Valida() {
		return nil, ErrOrigemInvalida
	}
	titulo = strings.TrimSpace(titulo)
	if len([]rune(titulo)) > tamanhoMaximoTitulo {
		return nil, ErrTituloLongo
	}
	if _, err := availability.NovoTimeBlock(inicioMinutos, fimMinutos); err != nil {
		return nil, err
	}

	agora := time.Now()
	return &Ocupacao{
		ID:            id,
		ProviderID:    providerID,
		Data:          data,
		InicioMinutos: inicioMinutos,
		FimMinutos:    fimMinutos,
		Titulo:        titulo,
		Origem:        origem,
		CriadoEm:      agora,
		AtualizadoEm:  agora,
	}, nil
}

// Valida informa se a origem é uma das reconhecidas pelo domínio. A coluna é
// VARCHAR sem CHECK justamente porque quem decide isso é aqui.
func (o Origem) Valida() bool {
	return o == OrigemManual
}

// Colide informa se a ocupação cruza o intervalo informado, sem considerar
// buffer.
//
// O buffer NÃO entra aqui de propósito: esta comparação serve para recusar uma
// ocupação sobre um agendamento existente, e ali o critério é sobreposição
// real. Já na oferta de horários, quem aplica o buffer dos dois lados é
// slot.Livres — um compromisso das 14h às 15h com buffer de 15 minutos também
// come as 13h45 e as 15h15, que é o comportamento desejado (deslocamento e
// preparação) e está documentado em docs/regra-de-negocio.md.
func (o *Ocupacao) Colide(inicioMinutos, fimMinutos int) bool {
	return o.InicioMinutos < fimMinutos && inicioMinutos < o.FimMinutos
}
