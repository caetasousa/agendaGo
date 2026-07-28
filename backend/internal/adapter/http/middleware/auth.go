// Package middleware contém os middlewares HTTP da aplicação.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"agendago/internal/adapter/http/handler"
	"agendago/internal/domain/membro"
	"agendago/internal/domain/session"
	ucauth "agendago/internal/usecase/auth"
)

type identidadeContextKey struct{}

// Auth valida a sessão do cookie e injeta a identidade do usuário no contexto
// da requisição. cookieSeguro decide o nome do cookie procurado — em produção
// ele leva o prefixo __Host- (ver handler.NomeCookieSessao).
type Auth struct {
	validar      *ucauth.ValidarSessaoUseCase
	cookieSeguro bool
}

// NovoAuth cria uma instância de Auth com o usecase de validação de sessão injetado.
func NovoAuth(validar *ucauth.ValidarSessaoUseCase, cookieSeguro bool) *Auth {
	return &Auth{validar: validar, cookieSeguro: cookieSeguro}
}

// Autenticar responde 401 se o cookie de sessão estiver ausente, ou se a
// sessão for inválida ou já tiver expirado. Caso contrário, injeta a
// identidade do usuário no contexto e segue para o próximo handler.
func (a *Auth) Autenticar(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(handler.NomeCookieSessao(a.cookieSeguro))
		if err != nil {
			responderNaoAutenticado(w)
			return
		}

		id, err := a.validar.Executar(cookie.Value)
		if err != nil {
			responderNaoAutenticado(w)
			return
		}

		ctx := context.WithValue(r.Context(), identidadeContextKey{}, *id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ExigirProvider responde 403 se o usuário autenticado não for prestador.
// Deve ser encadeado depois de Autenticar.
func ExigirProvider(next http.Handler) http.Handler {
	return exigirTipo(session.TipoProvider, next)
}

// ExigirClient responde 403 se o usuário autenticado não for cliente.
// Deve ser encadeado depois de Autenticar.
func ExigirClient(next http.Handler) http.Handler {
	return exigirTipo(session.TipoClient, next)
}

// ExigirAdmin responde 403 se o usuário autenticado não for administrador.
// Deve ser encadeado depois de Autenticar.
func ExigirAdmin(next http.Handler) http.Handler {
	return exigirTipo(session.TipoAdmin, next)
}

func exigirTipo(tipo session.TipoUsuario, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := IdentidadeDoContexto(r.Context())
		if !ok || id.Tipo != tipo {
			responderProibido(w, "acesso não permitido para este tipo de usuário")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ExigirGestaoDaAgenda responde 403 se o papel do usuário na agenda não lhe
// permitir operá-la. Deve ser encadeado depois de ExigirProvider.
//
// Hoje os dois papéis existentes passam, então este middleware não muda
// comportamento nenhum. Ele existe pela ordem em que as coisas dão errado: sem
// ele, um papel novo e mais restrito (um "leitor", por exemplo) nasceria com
// acesso a TODAS as rotas de `/providers/me/*` e só perderia esse acesso onde
// alguém lembrasse de checar. Com ele, o padrão é negar, e liberar passa a ser
// a decisão explícita — em um lugar só, no domínio.
func ExigirGestaoDaAgenda(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := IdentidadeDoContexto(r.Context())
		if !ok {
			responderNaoAutenticado(w)
			return
		}
		vinculo := membro.Membro{UsuarioID: id.UserID, ProviderID: id.ProviderID, Papel: id.Papel}
		if !vinculo.PodeGerenciarAgenda() {
			responderProibido(w, "seu papel nesta agenda não permite esta operação")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ExigirAdministracaoDaConta responde 403 se o usuário não for dono da agenda.
// Deve ser encadeado depois de ExigirProvider. Reservado para o que só o dono
// faz — encerrar a conta, convidar e remover membros.
func ExigirAdministracaoDaConta(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := IdentidadeDoContexto(r.Context())
		if !ok {
			responderNaoAutenticado(w)
			return
		}
		vinculo := membro.Membro{UsuarioID: id.UserID, ProviderID: id.ProviderID, Papel: id.Papel}
		if !vinculo.PodeAdministrarConta() {
			responderProibido(w, "apenas o dono da agenda pode fazer isso")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func responderProibido(w http.ResponseWriter, mensagem string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]string{"erro": mensagem})
}

// IdentidadeDoContexto recupera a identidade injetada pelo middleware Autenticar.
func IdentidadeDoContexto(ctx context.Context) (ucauth.Identidade, bool) {
	id, ok := ctx.Value(identidadeContextKey{}).(ucauth.Identidade)
	return id, ok
}

func responderNaoAutenticado(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"erro": "não autenticado"})
}
