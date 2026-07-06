// Package middleware contém os middlewares HTTP da aplicação.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"agendago/internal/adapter/http/handler"
	"agendago/internal/domain/session"
	ucauth "agendago/internal/usecase/auth"
)

type identidadeContextKey struct{}

// Auth valida a sessão do cookie e injeta a identidade do usuário no contexto da requisição.
type Auth struct {
	validar *ucauth.ValidarSessaoUseCase
}

// NovoAuth cria uma instância de Auth com o usecase de validação de sessão injetado.
func NovoAuth(validar *ucauth.ValidarSessaoUseCase) *Auth {
	return &Auth{validar: validar}
}

// Autenticar responde 401 se o cookie de sessão estiver ausente, ou se a
// sessão for inválida ou já tiver expirado. Caso contrário, injeta a
// identidade do usuário no contexto e segue para o próximo handler.
func (a *Auth) Autenticar(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(handler.NomeCookieSessao)
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

func exigirTipo(tipo session.TipoUsuario, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := IdentidadeDoContexto(r.Context())
		if !ok || id.Tipo != tipo {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"erro": "acesso não permitido para este tipo de usuário"})
			return
		}
		next.ServeHTTP(w, r)
	})
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
