package middleware

import "net/http"

// SemCache marca a resposta como não armazenável (Cache-Control: no-store).
// Sem esse cabeçalho o navegador pode guardar em disco, por heurística
// própria, o JSON das rotas autenticadas e das rotas por token — que carregam
// dados pessoais (contato de clientes, agendamentos). Vale para a resposta
// inteira, inclusive erros.
func SemCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
