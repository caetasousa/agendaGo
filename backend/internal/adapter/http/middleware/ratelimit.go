package middleware

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/go-chi/httprate"
)

// RespostaLimiteExcedido é o 429 do projeto: JSON com a chave "erro", como
// todas as outras respostas de erro da API — o padrão do httprate é texto
// puro, que o cliente HTTP do frontend não sabe ler.
func RespostaLimiteExcedido(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	json.NewEncoder(w).Encode(map[string]string{"erro": "muitas requisições; tente de novo em instantes"})
}

// ChavePorIP é a chave de rate limit por endereço do cliente.
//
// Existe no lugar de httprate.KeyByIP (deprecated na v0.16, porque não há
// origem de IP segura por padrão e a lib passou a exigir que cada projeto
// declare a sua): aqui a origem é explícita — r.RemoteAddr já foi reescrito
// por IPReal com o X-Real-IP que o Caddy define e o cliente não consegue
// forjar. CanonicalizeIP agrupa a faixa /64 de um cliente IPv6 numa chave só,
// senão trocar de endereço dentro da própria faixa zeraria o contador.
func ChavePorIP(r *http.Request) (string, error) {
	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	return httprate.CanonicalizeIP(ip), nil
}

// ChavePorSessao é a chave de rate limit por usuário autenticado — o teto que
// falta quando quem abusa já passou pelo login e o IP deixa de ser um
// identificador útil. Deve ser encadeada depois de Autenticar; sem identidade
// no contexto devolve erro, e o httprate corta a requisição (só aconteceria
// por engano de montagem do roteador).
func ChavePorSessao(r *http.Request) (string, error) {
	id, ok := IdentidadeDoContexto(r.Context())
	if !ok {
		return "", errors.New("rate limit por sessão exige o middleware Autenticar antes")
	}
	return string(id.Tipo) + ":" + id.UserID, nil
}
