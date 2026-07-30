//go:build swagger

package config_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"agendago/config"
)

// Par de swagger_desligado_test.go: com -tags=swagger a doc TEM que responder,
// senão o ambiente de desenvolvimento perdeu a ferramenta sem ninguém notar.
//
// Roda apenas com `go test -tags=swagger ./...`, que exige o pacote agendago/docs
// gerado por swag. Quem exercita isso de ponta a ponta é o job de E2E, que sobe
// o compose de desenvolvimento.
func TestSwaggerCompiladoResponde(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	r := config.NovoRouter()
	req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code == http.StatusNotFound {
		t.Error("esperava a doc montada com a build tag swagger, got: 404")
	}
}
