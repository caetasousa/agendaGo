package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agendago/internal/adapter/http/handler"
	"agendago/internal/adapter/http/middleware"
	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
	ucauth "agendago/internal/usecase/auth"

	"github.com/go-chi/chi/v5"
)

// novoRouterAuth monta um router chi completo com os repositórios de memória
// e um prestador e um cliente já cadastrados, espelhando o wiring de main.go.
func novoRouterAuth(t *testing.T) (*chi.Mux, *provider.Provider, *client.Client) {
	t.Helper()
	hasher := security.NovoHasherArgon2id()

	providerRepo := repository.NovoProviderMemoria()
	clientRepo := repository.NovoClientMemoria()
	sessionRepo := repository.NovoSessionMemoria()

	senhaHash, _ := hasher.Gerar("12345678")
	p, _ := provider.Novo("provider-1", "João Silva", "joao@email.com", "11999998888", senhaHash)
	providerRepo.Salvar(p)

	c, _ := client.NovoComConta("client-1", "Maria Silva", "maria@email.com", senhaHash)
	clientRepo.Salvar(c)

	loginProvider := ucauth.NovoLoginProviderUseCase(providerRepo, sessionRepo, hasher)
	loginClient := ucauth.NovoLoginClientUseCase(clientRepo, sessionRepo, hasher)
	logout := ucauth.NovoLogoutUseCase(sessionRepo)
	validarSessao := ucauth.NovoValidarSessaoUseCase(sessionRepo)
	perfil := ucauth.NovoPerfilUseCase(providerRepo, clientRepo, repository.NovoAdminMemoria())

	identidadeDoContexto := func(r *http.Request) (ucauth.Identidade, bool) {
		return middleware.IdentidadeDoContexto(r.Context())
	}
	authHandler := handler.NovoAuthHandler(loginProvider, loginClient, nil, logout, perfil, false, identidadeDoContexto)
	authMw := middleware.NovoAuth(validarSessao)

	r := chi.NewRouter()
	r.Post("/auth/provider/login", authHandler.LoginProvider)
	r.Post("/auth/client/login", authHandler.LoginClient)
	r.Post("/auth/logout", authHandler.Logout)
	r.Group(func(r chi.Router) {
		r.Use(authMw.Autenticar)
		r.Get("/auth/me", authHandler.Me)
		r.With(middleware.ExigirClient).Get("/somente-client", respondeOK)
		r.With(middleware.ExigirProvider).Get("/somente-provider", respondeOK)
	})

	return r, p, c
}

func respondeOK(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestLoginProviderHandler(t *testing.T) {
	t.Run("retorna 200 e cookie HttpOnly com credenciais corretas", func(t *testing.T) {
		r, _, _ := novoRouterAuth(t)
		body, _ := json.Marshal(map[string]string{"email": "joao@email.com", "senha": "12345678"})
		req := httptest.NewRequest(http.MethodPost, "/auth/provider/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d", rr.Code)
		}

		cookies := rr.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("esperava 1 cookie, got: %d", len(cookies))
		}
		cookie := cookies[0]
		if cookie.Name != handler.NomeCookieSessao {
			t.Errorf("esperava cookie %s, got: %s", handler.NomeCookieSessao, cookie.Name)
		}
		if !cookie.HttpOnly {
			t.Error("esperava cookie HttpOnly")
		}
		if cookie.SameSite != http.SameSiteLaxMode {
			t.Error("esperava SameSite=Lax")
		}
		if cookie.Path != "/" {
			t.Errorf("esperava Path=/, got: %s", cookie.Path)
		}
		maxAgeEsperado := int(ucauth.TTLSessao.Seconds())
		tolerancia := int(time.Minute.Seconds())
		if cookie.MaxAge <= 0 || cookie.MaxAge > maxAgeEsperado || cookie.MaxAge <= maxAgeEsperado-tolerancia {
			t.Errorf("esperava MaxAge próximo de %d segundos, got: %d", maxAgeEsperado, cookie.MaxAge)
		}
	})

	t.Run("retorna 401 com corpo genérico para senha incorreta", func(t *testing.T) {
		r, _, _ := novoRouterAuth(t)
		body, _ := json.Marshal(map[string]string{"email": "joao@email.com", "senha": "senha-errada"})
		req := httptest.NewRequest(http.MethodPost, "/auth/provider/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("esperava 401, got: %d", rr.Code)
		}
		var resp map[string]string
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["erro"] != "credenciais inválidas" {
			t.Errorf("esperava corpo genérico, got: %v", resp)
		}
	})

	t.Run("retorna 401 com o mesmo corpo genérico para email inexistente", func(t *testing.T) {
		r, _, _ := novoRouterAuth(t)
		body, _ := json.Marshal(map[string]string{"email": "inexistente@email.com", "senha": "12345678"})
		req := httptest.NewRequest(http.MethodPost, "/auth/provider/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("esperava 401, got: %d", rr.Code)
		}
	})
}

func TestLoginClientHandler(t *testing.T) {
	t.Run("retorna 200 e cookie HttpOnly com credenciais corretas", func(t *testing.T) {
		r, _, _ := novoRouterAuth(t)
		body, _ := json.Marshal(map[string]string{"email": "maria@email.com", "senha": "12345678"})
		req := httptest.NewRequest(http.MethodPost, "/auth/client/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d", rr.Code)
		}

		cookies := rr.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("esperava 1 cookie, got: %d", len(cookies))
		}
		if !cookies[0].HttpOnly {
			t.Error("esperava cookie HttpOnly")
		}

		var resp map[string]string
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["tipo"] != "client" {
			t.Errorf("esperava tipo 'client', got: %v", resp)
		}
	})

	t.Run("retorna 401 com corpo genérico para senha incorreta", func(t *testing.T) {
		r, _, _ := novoRouterAuth(t)
		body, _ := json.Marshal(map[string]string{"email": "maria@email.com", "senha": "senha-errada"})
		req := httptest.NewRequest(http.MethodPost, "/auth/client/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("esperava 401, got: %d", rr.Code)
		}
		var resp map[string]string
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["erro"] != "credenciais inválidas" {
			t.Errorf("esperava corpo genérico, got: %v", resp)
		}
	})
}

func TestMeHandler(t *testing.T) {
	t.Run("retorna 401 sem cookie de sessão", func(t *testing.T) {
		r, _, _ := novoRouterAuth(t)
		req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("esperava 401, got: %d", rr.Code)
		}
	})

	t.Run("devolve perfil do cliente após login de client", func(t *testing.T) {
		r, _, c := novoRouterAuth(t)

		body, _ := json.Marshal(map[string]string{"email": "maria@email.com", "senha": "12345678"})
		loginReq := httptest.NewRequest(http.MethodPost, "/auth/client/login", bytes.NewReader(body))
		loginReq.Header.Set("Content-Type", "application/json")
		loginRR := httptest.NewRecorder()
		r.ServeHTTP(loginRR, loginReq)
		cookie := loginRR.Result().Cookies()[0]

		meReq := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
		meReq.AddCookie(cookie)
		meRR := httptest.NewRecorder()
		r.ServeHTTP(meRR, meReq)

		if meRR.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d", meRR.Code)
		}
		var resp map[string]string
		json.NewDecoder(meRR.Body).Decode(&resp)
		if resp["tipo"] != "client" {
			t.Errorf("esperava tipo 'client', got: %v", resp)
		}
		if resp["id"] != c.ID {
			t.Errorf("esperava id %s, got: %s", c.ID, resp["id"])
		}
		if resp["email"] != "maria@email.com" {
			t.Errorf("esperava email 'maria@email.com', got: %s", resp["email"])
		}
	})
}

func TestFluxoLoginMeLogout(t *testing.T) {
	t.Run("login, me, logout e me novamente retorna 401", func(t *testing.T) {
		r, p, _ := novoRouterAuth(t)

		// login
		body, _ := json.Marshal(map[string]string{"email": "joao@email.com", "senha": "12345678"})
		loginReq := httptest.NewRequest(http.MethodPost, "/auth/provider/login", bytes.NewReader(body))
		loginReq.Header.Set("Content-Type", "application/json")
		loginRR := httptest.NewRecorder()
		r.ServeHTTP(loginRR, loginReq)
		if loginRR.Code != http.StatusOK {
			t.Fatalf("esperava 200 no login, got: %d", loginRR.Code)
		}
		cookie := loginRR.Result().Cookies()[0]

		// me autenticado
		meReq := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
		meReq.AddCookie(cookie)
		meRR := httptest.NewRecorder()
		r.ServeHTTP(meRR, meReq)
		if meRR.Code != http.StatusOK {
			t.Fatalf("esperava 200 em /auth/me, got: %d", meRR.Code)
		}
		var meResp map[string]string
		json.NewDecoder(meRR.Body).Decode(&meResp)
		if meResp["id"] != p.ID {
			t.Errorf("esperava id %s, got: %s", p.ID, meResp["id"])
		}
		if meResp["tipo"] != "provider" {
			t.Errorf("esperava tipo 'provider', got: %s", meResp["tipo"])
		}

		// logout
		logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		logoutReq.AddCookie(cookie)
		logoutRR := httptest.NewRecorder()
		r.ServeHTTP(logoutRR, logoutReq)
		if logoutRR.Code != http.StatusNoContent {
			t.Fatalf("esperava 204 no logout, got: %d", logoutRR.Code)
		}

		logoutCookies := logoutRR.Result().Cookies()
		if len(logoutCookies) != 1 {
			t.Fatalf("esperava 1 cookie de limpeza no logout, got: %d", len(logoutCookies))
		}
		if logoutCookies[0].Name != handler.NomeCookieSessao {
			t.Errorf("esperava cookie %s, got: %s", handler.NomeCookieSessao, logoutCookies[0].Name)
		}
		if logoutCookies[0].MaxAge >= 0 {
			t.Errorf("esperava MaxAge negativo (cookie expirado), got: %d", logoutCookies[0].MaxAge)
		}
		if logoutCookies[0].Value != "" {
			t.Errorf("esperava Value vazio no cookie de limpeza, got: %s", logoutCookies[0].Value)
		}

		// me após logout
		meReq2 := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
		meReq2.AddCookie(cookie)
		meRR2 := httptest.NewRecorder()
		r.ServeHTTP(meRR2, meReq2)
		if meRR2.Code != http.StatusUnauthorized {
			t.Errorf("esperava 401 após logout, got: %d", meRR2.Code)
		}
	})
}

func TestExigirClient(t *testing.T) {
	t.Run("retorna 403 para sessão de prestador", func(t *testing.T) {
		r, _, _ := novoRouterAuth(t)

		body, _ := json.Marshal(map[string]string{"email": "joao@email.com", "senha": "12345678"})
		loginReq := httptest.NewRequest(http.MethodPost, "/auth/provider/login", bytes.NewReader(body))
		loginReq.Header.Set("Content-Type", "application/json")
		loginRR := httptest.NewRecorder()
		r.ServeHTTP(loginRR, loginReq)
		cookie := loginRR.Result().Cookies()[0]

		req := httptest.NewRequest(http.MethodGet, "/somente-client", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("esperava 403, got: %d", rr.Code)
		}
	})

	t.Run("permite acesso para sessão de cliente", func(t *testing.T) {
		r, _, _ := novoRouterAuth(t)

		body, _ := json.Marshal(map[string]string{"email": "maria@email.com", "senha": "12345678"})
		loginReq := httptest.NewRequest(http.MethodPost, "/auth/client/login", bytes.NewReader(body))
		loginReq.Header.Set("Content-Type", "application/json")
		loginRR := httptest.NewRecorder()
		r.ServeHTTP(loginRR, loginReq)
		cookie := loginRR.Result().Cookies()[0]

		req := httptest.NewRequest(http.MethodGet, "/somente-client", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("esperava 200, got: %d", rr.Code)
		}
	})
}

func TestExigirProvider(t *testing.T) {
	t.Run("retorna 403 para sessão de cliente", func(t *testing.T) {
		r, _, _ := novoRouterAuth(t)

		body, _ := json.Marshal(map[string]string{"email": "maria@email.com", "senha": "12345678"})
		loginReq := httptest.NewRequest(http.MethodPost, "/auth/client/login", bytes.NewReader(body))
		loginReq.Header.Set("Content-Type", "application/json")
		loginRR := httptest.NewRecorder()
		r.ServeHTTP(loginRR, loginReq)
		cookie := loginRR.Result().Cookies()[0]

		req := httptest.NewRequest(http.MethodGet, "/somente-provider", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("esperava 403, got: %d", rr.Code)
		}
	})

	t.Run("permite acesso para sessão de prestador", func(t *testing.T) {
		r, _, _ := novoRouterAuth(t)

		body, _ := json.Marshal(map[string]string{"email": "joao@email.com", "senha": "12345678"})
		loginReq := httptest.NewRequest(http.MethodPost, "/auth/provider/login", bytes.NewReader(body))
		loginReq.Header.Set("Content-Type", "application/json")
		loginRR := httptest.NewRecorder()
		r.ServeHTTP(loginRR, loginReq)
		cookie := loginRR.Result().Cookies()[0]

		req := httptest.NewRequest(http.MethodGet, "/somente-provider", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("esperava 200, got: %d", rr.Code)
		}
	})
}
