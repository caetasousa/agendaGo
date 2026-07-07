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

// novoRouterAvailability monta um router chi com um prestador e um cliente já
// cadastrados e as rotas de disponibilidade protegidas por Autenticar +
// ExigirProvider, espelhando o wiring de main.go.
func novoRouterAvailability(t *testing.T) *chi.Mux {
	t.Helper()
	hasher := security.NovoHasherArgon2id()

	providerRepo := repository.NovoProviderMemoria()
	clientRepo := repository.NovoClientMemoria()
	sessionRepo := repository.NovoSessionMemoria()
	availabilityRepo := repository.NovoAvailabilityMemoria()

	senhaHash, _ := hasher.Gerar("12345678")
	p, _ := provider.Novo("provider-1", "João Silva", "joao@email.com", senhaHash)
	providerRepo.Salvar(p)

	c, _ := client.NovoComConta("client-1", "Maria Silva", "maria@email.com", senhaHash)
	clientRepo.Salvar(c)

	loginProvider := ucauth.NovoLoginProviderUseCase(providerRepo, sessionRepo, hasher)
	loginClient := ucauth.NovoLoginClientUseCase(clientRepo, sessionRepo, hasher)
	validarSessao := ucauth.NovoValidarSessaoUseCase(sessionRepo)

	identidadeDoContexto := func(req *http.Request) (ucauth.Identidade, bool) {
		return middleware.IdentidadeDoContexto(req.Context())
	}

	definirGradeSemanal := ucavailability.NovoDefinirGradeSemanalUseCase(availabilityRepo)
	consultarGradeSemanal := ucavailability.NovoConsultarGradeSemanalUseCase(availabilityRepo)
	criarExcecao := ucavailability.NovoCriarExcecaoUseCase(availabilityRepo)
	removerExcecao := ucavailability.NovoRemoverExcecaoUseCase(availabilityRepo)
	listarExcecoes := ucavailability.NovoListarExcecoesUseCase(availabilityRepo)

	availabilityHandler := handler.NovoAvailabilityHandler(
		definirGradeSemanal, consultarGradeSemanal, criarExcecao, removerExcecao, listarExcecoes,
		identidadeDoContexto,
	)
	authHandler := handler.NovoAuthHandler(loginProvider, loginClient, nil, nil, false, identidadeDoContexto)
	authMw := middleware.NovoAuth(validarSessao)

	router := chi.NewRouter()
	router.Post("/auth/provider/login", authHandler.LoginProvider)
	router.Post("/auth/client/login", authHandler.LoginClient)
	router.Group(func(router chi.Router) {
		router.Use(authMw.Autenticar)
		router.Use(middleware.ExigirProvider)
		router.Get("/providers/me/disponibilidade", availabilityHandler.ConsultarGradeSemanal)
		router.Put("/providers/me/disponibilidade", availabilityHandler.DefinirGradeSemanal)
		router.Get("/providers/me/excecoes", availabilityHandler.ListarExcecoes)
		router.Post("/providers/me/excecoes", availabilityHandler.CriarExcecao)
		router.Delete("/providers/me/excecoes/{id}", availabilityHandler.RemoverExcecao)
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

func TestHandlerDisponibilidade(t *testing.T) {
	t.Run("PUT retorna 200 e persiste a grade para sessão de prestador", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body := map[string]any{
			"dias": []map[string]any{
				{"diaSemana": 1, "blocos": []map[string]any{{"inicioMinutos": 480, "fimMinutos": 720}}},
			},
		}
		rr := requisicaoComCookie(t, r, http.MethodPut, "/providers/me/disponibilidade", body, cookie)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d, body: %s", rr.Code, rr.Body.String())
		}

		var resp map[string]any
		json.NewDecoder(rr.Body).Decode(&resp)
		dias := resp["dias"].([]any)
		if len(dias) != 1 {
			t.Errorf("esperava 1 dia configurado, got: %v", dias)
		}
	})

	t.Run("PUT retorna 400 quando bloco tem granularidade inválida", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body := map[string]any{
			"dias": []map[string]any{
				{"diaSemana": 1, "blocos": []map[string]any{{"inicioMinutos": 485, "fimMinutos": 720}}},
			},
		}
		rr := requisicaoComCookie(t, r, http.MethodPut, "/providers/me/disponibilidade", body, cookie)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("PUT retorna 401 sem cookie de sessão", func(t *testing.T) {
		r := novoRouterAvailability(t)
		rr := requisicaoComCookie(t, r, http.MethodPut, "/providers/me/disponibilidade", map[string]any{"dias": []any{}}, nil)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("esperava 401, got: %d", rr.Code)
		}
	})

	t.Run("PUT retorna 403 quando cliente tenta definir a grade", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/client/login", "maria@email.com", "12345678")

		rr := requisicaoComCookie(t, r, http.MethodPut, "/providers/me/disponibilidade", map[string]any{"dias": []any{}}, cookie)
		if rr.Code != http.StatusForbidden {
			t.Errorf("esperava 403, got: %d", rr.Code)
		}
	})

	t.Run("GET retorna 200 com grade vazia para prestador novo", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		rr := requisicaoComCookie(t, r, http.MethodGet, "/providers/me/disponibilidade", nil, cookie)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d", rr.Code)
		}
		var resp map[string]any
		json.NewDecoder(rr.Body).Decode(&resp)
		dias := resp["dias"].([]any)
		if len(dias) != 0 {
			t.Errorf("esperava grade vazia, got: %v", dias)
		}
	})
}

func TestHandlerExcecoes(t *testing.T) {
	t.Run("POST cria bloqueio com sucesso", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body := map[string]any{"data": "2026-08-10", "tipo": "bloqueio", "blocos": []any{}}
		rr := requisicaoComCookie(t, r, http.MethodPost, "/providers/me/excecoes", body, cookie)
		if rr.Code != http.StatusCreated {
			t.Fatalf("esperava 201, got: %d, body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("POST cria extra com blocos", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body := map[string]any{
			"data": "2026-08-11", "tipo": "extra",
			"blocos": []map[string]any{{"inicioMinutos": 600, "fimMinutos": 660}},
		}
		rr := requisicaoComCookie(t, r, http.MethodPost, "/providers/me/excecoes", body, cookie)
		if rr.Code != http.StatusCreated {
			t.Fatalf("esperava 201, got: %d, body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("POST retorna 400 quando tipo é inválido", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body := map[string]any{"data": "2026-08-12", "tipo": "feriado", "blocos": []any{}}
		rr := requisicaoComCookie(t, r, http.MethodPost, "/providers/me/excecoes", body, cookie)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("POST retorna 400 quando data é mal formatada", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body := map[string]any{"data": "10/08/2026", "tipo": "bloqueio", "blocos": []any{}}
		rr := requisicaoComCookie(t, r, http.MethodPost, "/providers/me/excecoes", body, cookie)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("POST retorna 409 quando já existe exceção para a data", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body := map[string]any{"data": "2026-08-20", "tipo": "bloqueio", "blocos": []any{}}
		requisicaoComCookie(t, r, http.MethodPost, "/providers/me/excecoes", body, cookie)
		rr := requisicaoComCookie(t, r, http.MethodPost, "/providers/me/excecoes", body, cookie)
		if rr.Code != http.StatusConflict {
			t.Errorf("esperava 409, got: %d", rr.Code)
		}
	})

	t.Run("GET lista as exceções do prestador", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body := map[string]any{"data": "2026-08-25", "tipo": "bloqueio", "blocos": []any{}}
		requisicaoComCookie(t, r, http.MethodPost, "/providers/me/excecoes", body, cookie)

		rr := requisicaoComCookie(t, r, http.MethodGet, "/providers/me/excecoes", nil, cookie)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d", rr.Code)
		}
		var resp map[string]any
		json.NewDecoder(rr.Body).Decode(&resp)
		excecoes := resp["excecoes"].([]any)
		if len(excecoes) != 1 {
			t.Errorf("esperava 1 exceção, got: %v", excecoes)
		}
	})

	t.Run("DELETE remove exceção com 204", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body := map[string]any{"data": "2026-08-30", "tipo": "bloqueio", "blocos": []any{}}
		criarRR := requisicaoComCookie(t, r, http.MethodPost, "/providers/me/excecoes", body, cookie)
		var criada map[string]any
		json.NewDecoder(criarRR.Body).Decode(&criada)

		rr := requisicaoComCookie(t, r, http.MethodDelete, "/providers/me/excecoes/"+criada["id"].(string), nil, cookie)
		if rr.Code != http.StatusNoContent {
			t.Errorf("esperava 204, got: %d", rr.Code)
		}
	})

	t.Run("DELETE retorna 404 para exceção inexistente", func(t *testing.T) {
		r := novoRouterAvailability(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		rr := requisicaoComCookie(t, r, http.MethodDelete, "/providers/me/excecoes/id-fantasma", nil, cookie)
		if rr.Code != http.StatusNotFound {
			t.Errorf("esperava 404, got: %d", rr.Code)
		}
	})

	t.Run("POST retorna 401 sem cookie e 403 para cliente", func(t *testing.T) {
		r := novoRouterAvailability(t)
		body := map[string]any{"data": "2026-09-01", "tipo": "bloqueio", "blocos": []any{}}

		rrSemCookie := requisicaoComCookie(t, r, http.MethodPost, "/providers/me/excecoes", body, nil)
		if rrSemCookie.Code != http.StatusUnauthorized {
			t.Errorf("esperava 401, got: %d", rrSemCookie.Code)
		}

		cookieCliente := loginEObterCookie(t, r, "/auth/client/login", "maria@email.com", "12345678")
		rrCliente := requisicaoComCookie(t, r, http.MethodPost, "/providers/me/excecoes", body, cookieCliente)
		if rrCliente.Code != http.StatusForbidden {
			t.Errorf("esperava 403, got: %d", rrCliente.Code)
		}
	})
}
