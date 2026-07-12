package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agendago/internal/adapter/http/handler"
	"agendago/internal/adapter/http/middleware"
	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
	ucauth "agendago/internal/usecase/auth"
	ucavailability "agendago/internal/usecase/availability"

	"github.com/go-chi/chi/v5"
)

// novoRouterAvailability monta um router chi com um prestador (agenda ativa) e
// um cliente já cadastrados e as rotas de disponibilidade protegidas por
// Autenticar + ExigirProvider, espelhando o wiring de main.go.
func novoRouterAvailability(t *testing.T) *chi.Mux {
	t.Helper()
	hasher := security.NovoHasherArgon2id()

	providerRepo := repository.NovoProviderMemoria()
	clientRepo := repository.NovoClientMemoria()
	sessionRepo := repository.NovoSessionMemoria()
	availabilityRepo := repository.NovoAvailabilityMemoria()

	senhaHash, _ := hasher.Gerar("12345678")
	p, _ := provider.Novo("provider-1", "João Silva", "joao@email.com", senhaHash)
	p.AtivarAgenda()
	providerRepo.Salvar(p)

	c, _ := client.NovoComConta("client-1", "Maria Silva", "maria@email.com", senhaHash)
	clientRepo.Salvar(c)

	loginProvider := ucauth.NovoLoginProviderUseCase(providerRepo, sessionRepo, hasher)
	loginClient := ucauth.NovoLoginClientUseCase(clientRepo, sessionRepo, hasher)
	validarSessao := ucauth.NovoValidarSessaoUseCase(sessionRepo)

	identidadeDoContexto := func(req *http.Request) (ucauth.Identidade, bool) {
		return middleware.IdentidadeDoContexto(req.Context())
	}

	consultarAgenda := ucavailability.NovoConsultarAgendaUseCase(availabilityRepo, providerRepo)
	definirDia := ucavailability.NovoDefinirDiaUseCase(availabilityRepo)
	removerDia := ucavailability.NovoRemoverDiaUseCase(availabilityRepo)

	availabilityHandler := handler.NovoAvailabilityHandler(consultarAgenda, definirDia, removerDia, identidadeDoContexto)
	authHandler := handler.NovoAuthHandler(loginProvider, loginClient, nil, nil, nil, false, identidadeDoContexto)
	authMw := middleware.NovoAuth(validarSessao)

	router := chi.NewRouter()
	router.Post("/auth/provider/login", authHandler.LoginProvider)
	router.Post("/auth/client/login", authHandler.LoginClient)
	router.Group(func(router chi.Router) {
		router.Use(authMw.Autenticar)
		router.Use(middleware.ExigirProvider)
		router.Get("/providers/me/agenda", availabilityHandler.ConsultarAgenda)
		router.Put("/providers/me/dias/{data}", availabilityHandler.DefinirDia)
		router.Delete("/providers/me/dias/{data}", availabilityHandler.RemoverDia)
	})

	return router
}

func requisicaoComCookie(t *testing.T, r *chi.Mux, method, rota string, corpo any, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var body *bytes.Reader
	if corpo != nil {
		b, _ := json.Marshal(corpo)
		body = bytes.NewReader(b)
	} else {
		body = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, rota, body)
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestHandlerAgenda(t *testing.T) {
	t.Run("GET resolve o período com padrão e definições próprias", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		// 2026-08-10 (segunda) vira bloqueio; o resto da semana fica no padrão
		body := map[string]any{"tipo": "bloqueio", "blocos": []any{}}
		requisicaoComCookie(t, r, http.MethodPut, "/providers/me/dias/2026-08-10", body, cookie)

		rr := requisicaoComCookie(t, r, http.MethodGet, "/providers/me/agenda?de=2026-08-10&ate=2026-08-16", nil, cookie)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d, body: %s", rr.Code, rr.Body.String())
		}

		var resp map[string]any
		json.NewDecoder(rr.Body).Decode(&resp)
		dias := resp["dias"].([]any)
		if len(dias) != 7 {
			t.Fatalf("esperava 7 dias, got: %d", len(dias))
		}
		segunda := dias[0].(map[string]any)
		if segunda["origem"] != "bloqueio" {
			t.Errorf("esperava segunda com origem bloqueio, got: %v", segunda)
		}
		terca := dias[1].(map[string]any)
		if terca["origem"] != "padrao" || len(terca["blocos"].([]any)) != 2 {
			t.Errorf("esperava terça no padrão comercial, got: %v", terca)
		}
	})

	t.Run("GET retorna 400 sem parâmetros de período", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		rr := requisicaoComCookie(t, r, http.MethodGet, "/providers/me/agenda", nil, cookie)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("GET retorna 400 para período invertido", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		rr := requisicaoComCookie(t, r, http.MethodGet, "/providers/me/agenda?de=2026-08-16&ate=2026-08-10", nil, cookie)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("GET retorna 401 sem cookie e 403 para cliente", func(t *testing.T) {
		r := novoRouterAvailability(t)

		rrSemCookie := requisicaoComCookie(t, r, http.MethodGet, "/providers/me/agenda?de=2026-08-10&ate=2026-08-16", nil, nil)
		if rrSemCookie.Code != http.StatusUnauthorized {
			t.Errorf("esperava 401, got: %d", rrSemCookie.Code)
		}

		cookieCliente := loginEObterCookie(t, r, "/auth/client/login", "maria@email.com", "12345678")
		rrCliente := requisicaoComCookie(t, r, http.MethodGet, "/providers/me/agenda?de=2026-08-10&ate=2026-08-16", nil, cookieCliente)
		if rrCliente.Code != http.StatusForbidden {
			t.Errorf("esperava 403, got: %d", rrCliente.Code)
		}
	})
}

func TestHandlerDefinirDia(t *testing.T) {
	t.Run("PUT define bloqueio com 200", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body := map[string]any{"tipo": "bloqueio", "blocos": []any{}}
		rr := requisicaoComCookie(t, r, http.MethodPut, "/providers/me/dias/2026-08-10", body, cookie)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d, body: %s", rr.Code, rr.Body.String())
		}

		var resp map[string]any
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["origem"] != "bloqueio" || resp["data"] != "2026-08-10" {
			t.Errorf("esperava dia bloqueado em 2026-08-10, got: %v", resp)
		}
	})

	t.Run("PUT define extra com blocos", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body := map[string]any{
			"tipo":   "extra",
			"blocos": []map[string]any{{"inicioMinutos": 600, "fimMinutos": 660}},
		}
		rr := requisicaoComCookie(t, r, http.MethodPut, "/providers/me/dias/2026-08-11", body, cookie)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d, body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("PUT na mesma data substitui a definição (upsert, sem 409)", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		requisicaoComCookie(t, r, http.MethodPut, "/providers/me/dias/2026-08-20",
			map[string]any{"tipo": "bloqueio", "blocos": []any{}}, cookie)
		rr := requisicaoComCookie(t, r, http.MethodPut, "/providers/me/dias/2026-08-20",
			map[string]any{"tipo": "extra", "blocos": []map[string]any{{"inicioMinutos": 480, "fimMinutos": 720}}}, cookie)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200 no upsert, got: %d, body: %s", rr.Code, rr.Body.String())
		}

		var resp map[string]any
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["origem"] != "extra" {
			t.Errorf("esperava origem extra após substituir, got: %v", resp)
		}
	})

	t.Run("PUT retorna 400 quando tipo é inválido", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body := map[string]any{"tipo": "feriado", "blocos": []any{}}
		rr := requisicaoComCookie(t, r, http.MethodPut, "/providers/me/dias/2026-08-12", body, cookie)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("PUT retorna 400 quando data é mal formatada", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body := map[string]any{"tipo": "bloqueio", "blocos": []any{}}
		rr := requisicaoComCookie(t, r, http.MethodPut, "/providers/me/dias/10-08-2026", body, cookie)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("PUT retorna 400 quando bloco tem granularidade inválida", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body := map[string]any{
			"tipo":   "extra",
			"blocos": []map[string]any{{"inicioMinutos": 485, "fimMinutos": 720}},
		}
		rr := requisicaoComCookie(t, r, http.MethodPut, "/providers/me/dias/2026-08-13", body, cookie)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("PUT retorna 401 sem cookie e 403 para cliente", func(t *testing.T) {
		r := novoRouterAvailability(t)
		body := map[string]any{"tipo": "bloqueio", "blocos": []any{}}

		rrSemCookie := requisicaoComCookie(t, r, http.MethodPut, "/providers/me/dias/2026-09-01", body, nil)
		if rrSemCookie.Code != http.StatusUnauthorized {
			t.Errorf("esperava 401, got: %d", rrSemCookie.Code)
		}

		cookieCliente := loginEObterCookie(t, r, "/auth/client/login", "maria@email.com", "12345678")
		rrCliente := requisicaoComCookie(t, r, http.MethodPut, "/providers/me/dias/2026-09-01", body, cookieCliente)
		if rrCliente.Code != http.StatusForbidden {
			t.Errorf("esperava 403, got: %d", rrCliente.Code)
		}
	})
}

func TestHandlerRemoverDia(t *testing.T) {
	t.Run("DELETE remove a definição com 204", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body := map[string]any{"tipo": "bloqueio", "blocos": []any{}}
		requisicaoComCookie(t, r, http.MethodPut, "/providers/me/dias/2026-08-30", body, cookie)

		rr := requisicaoComCookie(t, r, http.MethodDelete, "/providers/me/dias/2026-08-30", nil, cookie)
		if rr.Code != http.StatusNoContent {
			t.Errorf("esperava 204, got: %d", rr.Code)
		}
	})

	t.Run("DELETE retorna 404 para data sem definição própria", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		rr := requisicaoComCookie(t, r, http.MethodDelete, "/providers/me/dias/2030-01-01", nil, cookie)
		if rr.Code != http.StatusNotFound {
			t.Errorf("esperava 404, got: %d", rr.Code)
		}
	})

	t.Run("DELETE retorna 400 para data mal formatada", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		rr := requisicaoComCookie(t, r, http.MethodDelete, "/providers/me/dias/hoje", nil, cookie)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})
}
