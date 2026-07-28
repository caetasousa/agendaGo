package auth

import (
	"time"

	"agendago/internal/domain/admin"
	"agendago/internal/domain/client"
	"agendago/internal/domain/membro"
	"agendago/internal/domain/passwordreset"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/session"
	"agendago/internal/domain/usuario"
)

type repositorioSessao interface {
	Salvar(s *session.Session) error
	BuscarPorTokenHash(hash string) (*session.Session, error)
	Remover(hash string) error
	RemoverDoUsuario(userID string) error
	RemoverExpiradas() error
}

// buscadorProvider busca a AGENDA. Identidade (email, senha, telefone,
// situação) mora em usuarios — por isso não há busca por email aqui.
type buscadorProvider interface {
	BuscarPorID(id string) (*provider.Provider, error)
}

// buscadorUsuario busca a IDENTIDADE de quem loga pelo lado prestador.
type buscadorUsuario interface {
	BuscarPorEmail(email string) (*usuario.Usuario, error)
	BuscarPorID(id string) (*usuario.Usuario, error)
}

// buscadorMembro resolve qual agenda um usuário opera.
type buscadorMembro interface {
	BuscarPorUsuario(usuarioID string) (*membro.Membro, error)
}

// contaUsuario busca usuários e persiste a troca de senha.
type contaUsuario interface {
	buscadorUsuario
	AtualizarSenha(id, senhaHash string) error
}

type buscadorClient interface {
	BuscarPorEmail(email string) (*client.Client, error)
	BuscarPorID(id string) (*client.Client, error)
}

type buscadorAdmin interface {
	BuscarPorEmail(email string) (*admin.Admin, error)
	BuscarPorID(id string) (*admin.Admin, error)
}

// contaClient busca clientes por email e persiste a troca de senha.
type contaClient interface {
	buscadorClient
	AtualizarSenha(id, senhaHash string) error
}

// repositorioResetSenha persiste e consome tokens de recuperação de senha.
// Consumir apaga o token ao lê-lo, garantindo uso único.
type repositorioResetSenha interface {
	Salvar(t *passwordreset.Token) error
	Consumir(tokenHash string) (*passwordreset.Token, error)
	RemoverDoUsuario(userID string) error
	RemoverExpirados() error
}

// enviadorRecuperacao envia o email com o link de redefinição de senha.
type enviadorRecuperacao interface {
	EnviarRecuperacaoSenha(email, nome, token string, expiraEm time.Time)
}

// hasherSenha gera e verifica hashes de senha.
type hasherSenha interface {
	Gerar(senha string) (string, error)
	Verificar(senha, hash string) (bool, error)
}
