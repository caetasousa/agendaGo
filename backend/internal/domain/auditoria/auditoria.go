// Trilha de auditoria: registro imutável de quem fez o quê, com quem e quando.
//
// Só ações sensíveis entram — moderação de conta e exclusão de dados. Registrar
// tudo transformaria a trilha num segundo log de acesso, e o que se quer aqui é
// poder responder "quem baniu este prestador, e quando" sem garimpar.
package auditoria

import (
	"errors"
	"time"
)

// Acao identifica o que foi feito. String, e não enum fechado, porque a lista
// cresce a cada ação sensível nova e um valor desconhecido lido do banco não
// pode quebrar a leitura de uma trilha histórica.
type Acao string

const (
	// AcaoBanirPrestador registra a suspensão de um prestador pelo admin.
	AcaoBanirPrestador Acao = "banir_prestador"
	// AcaoReativarPrestador registra a reativação de um prestador pelo admin.
	AcaoReativarPrestador Acao = "reativar_prestador"
	// AcaoBanirCliente registra a suspensão de um cliente pelo admin.
	AcaoBanirCliente Acao = "banir_cliente"
	// AcaoReativarCliente registra a reativação de um cliente pelo admin.
	AcaoReativarCliente Acao = "reativar_cliente"
	// AcaoAnonimizarCliente registra o atendimento de um pedido de exclusão.
	AcaoAnonimizarCliente Acao = "anonimizar_cliente"
	// AcaoExportarDados registra o atendimento de um pedido de portabilidade.
	AcaoExportarDados Acao = "exportar_dados"
)

// TipoAtor e TipoAlvo identificam de que lado da ação está cada parte.
const (
	TipoAdmin    = "admin"
	TipoCliente  = "client"
	TipoUsuario  = "usuario"
	TipoProvider = "provider"
)

var (
	// ErrAtorObrigatorio é retornado quando o registro não identifica quem agiu.
	ErrAtorObrigatorio = errors.New("ator é obrigatório")
	// ErrAlvoObrigatorio é retornado quando o registro não identifica sobre quem.
	ErrAlvoObrigatorio = errors.New("alvo é obrigatório")
	// ErrAcaoObrigatoria é retornado quando o registro não diz o que foi feito.
	ErrAcaoObrigatoria = errors.New("ação é obrigatória")
)

// Registro é uma entrada da trilha. Depois de criado nunca é alterado — não há
// mutador neste tipo, e o repositório não expõe atualização nem remoção.
type Registro struct {
	ID       string
	AtorID   string
	AtorTipo string
	Acao     Acao
	AlvoTipo string
	AlvoID   string
	Detalhe  map[string]any
	CriadoEm time.Time
}

// Novo cria uma entrada da trilha.
//
// Detalhe é livre, mas NÃO deve carregar dado pessoal: o alvo é identificado
// por id. Guardar nome ou email aqui recriaria, na trilha, exatamente o que a
// anonimização acabou de apagar do cadastro.
func Novo(id, atorID, atorTipo string, acao Acao, alvoTipo, alvoID string, detalhe map[string]any) (*Registro, error) {
	if atorID == "" || atorTipo == "" {
		return nil, ErrAtorObrigatorio
	}
	if alvoID == "" || alvoTipo == "" {
		return nil, ErrAlvoObrigatorio
	}
	if acao == "" {
		return nil, ErrAcaoObrigatoria
	}
	return &Registro{
		ID:       id,
		AtorID:   atorID,
		AtorTipo: atorTipo,
		Acao:     acao,
		AlvoTipo: alvoTipo,
		AlvoID:   alvoID,
		Detalhe:  detalhe,
		CriadoEm: time.Now(),
	}, nil
}
