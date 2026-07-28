package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agendago/internal/adapter/http/handler"
	"agendago/internal/adapter/security"
	ucauth "agendago/internal/usecase/auth"
	"agendago/test/repository/memoria"
)

// Em produção o cookie de sessão leva o prefixo __Host-, que o navegador só
// aceita com Secure, Path=/ e sem Domain. É o que impede um subdomínio
// comprometido de sobrescrever a sessão do domínio principal. Em
// desenvolvimento o prefixo não pode existir: sem HTTPS não há Secure, e o
// navegador recusaria o cookie inteiro.
func TestNomeDoCookieDeSessao(t *testing.T) {
	logar := func(t *testing.T, cookieSeguro bool) *http.Cookie {
		t.Helper()
		hasher := security.NovoHasherArgon2id()
		usuarios, membros, providers := fakesDePrestador()
		sessionRepo := memoria.NovoSessionMemoria()

		senhaHash, _ := hasher.Gerar("12345678")
		_, p := criarPrestador(usuarios, membros, providers, "provider-1", "João Silva", "joao@email.com", "11999998888", senhaHash)
		providers.Salvar(p)

		h := handler.NovoAuthHandler(
			ucauth.NovoLoginProviderUseCase(usuarios, membros, providers, sessionRepo, hasher),
			nil, nil, nil, nil, cookieSeguro, nil, nil,
		)

		body, _ := json.Marshal(map[string]string{"email": "joao@email.com", "senha": "12345678"})
		req := httptest.NewRequest(http.MethodPost, "/auth/provider/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.LoginProvider(rr, req)

		cookies := rr.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("esperava 1 cookie, got: %d", len(cookies))
		}
		return cookies[0]
	}

	t.Run("produção usa o prefixo __Host- com os atributos que ele exige", func(t *testing.T) {
		cookie := logar(t, true)

		if esperado := handler.NomeCookieSessao(true); cookie.Name != esperado {
			t.Errorf("esperava cookie %q, got: %q", esperado, cookie.Name)
		}
		if cookie.Name != "__Host-agendago_session" {
			t.Errorf("esperava o prefixo __Host-, got: %q", cookie.Name)
		}
		// os três atributos sem os quais o navegador ignora um cookie __Host-
		if !cookie.Secure {
			t.Error("__Host- exige Secure")
		}
		if cookie.Path != "/" {
			t.Errorf("__Host- exige Path=/, got: %q", cookie.Path)
		}
		if cookie.Domain != "" {
			t.Errorf("__Host- não admite Domain, got: %q", cookie.Domain)
		}
	})

	t.Run("desenvolvimento usa o nome simples, sem Secure", func(t *testing.T) {
		cookie := logar(t, false)

		if cookie.Name != "agendago_session" {
			t.Errorf("esperava agendago_session em dev, got: %q", cookie.Name)
		}
		if cookie.Secure {
			t.Error("não esperava Secure em desenvolvimento (http://localhost)")
		}
	})
}
