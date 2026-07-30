//go:build !swagger

package config_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"agendago/config"
)

// A doc do swagger publica a superfície inteira da API e não pode existir em
// produção. Antes essa garantia era por ambiente (`if !EhProducao()`); agora é
// por compilação, e este teste é o que a fixa: no build padrão — o que vai para
// produção — a rota não existe em NENHUM ambiente.
//
// Por que testar com APP_ENV=development, que seria o caso mais permissivo: se
// alguém remontar a rota atrás de uma condição de ambiente, é aqui que quebra.
// O par deste teste está em swagger_ligado_test.go, que roda com -tags=swagger.
//
// O Caddy também devolve 404 nesse caminho, mas isso é configuração de
// infraestrutura — o que se garante aqui é que a própria API não serve a rota,
// mesmo se for exposta por outro proxy ou alcançada pela rede interna.
func TestSwaggerNaoCompiladoNuncaResponde(t *testing.T) {
	casos := []string{"production", "development", ""}

	for _, ambiente := range casos {
		nome := ambiente
		if nome == "" {
			nome = "sem APP_ENV"
		}
		t.Run(nome, func(t *testing.T) {
			t.Setenv("APP_ENV", ambiente)
			r := config.NovoRouter()
			req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			if resp.Code != http.StatusNotFound {
				t.Errorf("esperava 404 sem a build tag, got: %d", resp.Code)
			}
		})
	}
}
