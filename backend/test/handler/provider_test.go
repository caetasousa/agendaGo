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
	ucprovider "agendago/internal/usecase/provider"

	"github.com/go-chi/chi/v5"
)

// identidadeAusente simula uma requisição sem identidade no contexto —
// suficiente para os testes que só exercitam Cadastrar (rota pública).
func identidadeAusente(r *http.Request) (ucauth.Identidade, bool) {
	return ucauth.Identidade{}, false
}

func novoHandler() *handler.ProviderHandler {
	repo := repository.NovoProviderMemoria()
	cadastrar := ucprovider.NovoCadastrarUseCase(repo, repository.NovoClientMemoria(), security.NovoHasherArgon2id())
	atualizarPreferencias := ucprovider.NovoAtualizarPreferenciasUseCase(repo)
	listar := ucprovider.NovoListarUseCase(repo)
	buscarResumo := ucprovider.NovoBuscarResumoUseCase(repo)
	return handler.NovoProviderHandler(cadastrar, atualizarPreferencias, listar, buscarResumo, identidadeAusente)
}

func fazerRequisicao(t *testing.T, h *handler.ProviderHandler, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/providers", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Cadastrar(rr, req)
	return rr
}

// novoRouterPreferencias monta um router chi com um prestador e um cliente já
// cadastrados e a rota PUT /providers/me/preferencias protegida por
// Autenticar + ExigirProvider, espelhando o wiring de main.go.
func novoRouterPreferencias(t *testing.T) (r *chi.Mux, providerID string) {
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
	validarSessao := ucauth.NovoValidarSessaoUseCase(sessionRepo)

	identidadeDoContexto := func(req *http.Request) (ucauth.Identidade, bool) {
		return middleware.IdentidadeDoContexto(req.Context())
	}
	atualizarPreferencias := ucprovider.NovoAtualizarPreferenciasUseCase(providerRepo)
	listar := ucprovider.NovoListarUseCase(providerRepo)
	buscarResumo := ucprovider.NovoBuscarResumoUseCase(providerRepo)
	providerHandler := handler.NovoProviderHandler(nil, atualizarPreferencias, listar, buscarResumo, identidadeDoContexto)
	authHandler := handler.NovoAuthHandler(loginProvider, loginClient, nil, nil, nil, false, identidadeDoContexto)
	authMw := middleware.NovoAuth(validarSessao)

	router := chi.NewRouter()
	router.Post("/auth/provider/login", authHandler.LoginProvider)
	router.Post("/auth/client/login", authHandler.LoginClient)
	router.Group(func(router chi.Router) {
		router.Use(authMw.Autenticar)
		router.Use(middleware.ExigirProvider)
		router.Put("/providers/me/preferencias", providerHandler.AtualizarPreferencias)
	})

	return router, p.ID
}

func loginEObterCookie(t *testing.T, r *chi.Mux, rota, email, senha string) *http.Cookie {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "senha": senha})
	req := httptest.NewRequest(http.MethodPost, rota, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("esperava 200 no login, got: %d", rr.Code)
	}
	return rr.Result().Cookies()[0]
}

func TestHandlerCadastrarProvider(t *testing.T) {
	t.Run("retorna 201 e ID do provider criado quando dados são válidos", func(t *testing.T) {
		rr := fazerRequisicao(t, novoHandler(), map[string]string{
			"nome":     "João Silva",
			"email":    "joao@email.com",
			"telefone": "11999998888",
			"senha":    "12345678",
		})
		if rr.Code != http.StatusCreated {
			t.Errorf("esperava 201, got: %d", rr.Code)
		}
		var resp map[string]string
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["id"] == "" {
			t.Error("ID não deve ser vazio na resposta")
		}
	})

	t.Run("retorna 400 quando email não tem formato válido", func(t *testing.T) {
		rr := fazerRequisicao(t, novoHandler(), map[string]string{
			"nome":     "João Silva",
			"email":    "emailinvalido",
			"telefone": "11999998888",
			"senha":    "12345678",
		})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("retorna 400 quando senha tem menos de 8 caracteres", func(t *testing.T) {
		rr := fazerRequisicao(t, novoHandler(), map[string]string{
			"nome":     "João Silva",
			"email":    "joao@email.com",
			"telefone": "11999998888",
			"senha":    "123",
		})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("retorna 409 quando email já está cadastrado", func(t *testing.T) {
		h := novoHandler()
		body := map[string]string{
			"nome":     "João Silva",
			"email":    "joao@email.com",
			"telefone": "11999998888",
			"senha":    "12345678",
		}
		fazerRequisicao(t, h, body)
		rr := fazerRequisicao(t, h, body)
		if rr.Code != http.StatusConflict {
			t.Errorf("esperava 409, got: %d", rr.Code)
		}
	})

	t.Run("retorna 400 quando body não é um JSON válido", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/providers", bytes.NewBufferString("não é json"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		novoHandler().Cadastrar(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})
}

func TestHandlerAtualizarPreferencias(t *testing.T) {
	t.Run("retorna 200 e persiste as preferências para sessão de prestador", func(t *testing.T) {
		r, _ := novoRouterPreferencias(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body, _ := json.Marshal(map[string]any{"telefone": "11999998888", "aceitaAgendamentos": true, "descansoMinutos": 15, "duracaoAtendimentoMinutos": 60})
		req := httptest.NewRequest(http.MethodPut, "/providers/me/preferencias", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d", rr.Code)
		}
		var resp map[string]any
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["aceitaAgendamentos"] != true {
			t.Errorf("esperava aceitaAgendamentos true, got: %v", resp["aceitaAgendamentos"])
		}
		if resp["descansoMinutos"] != float64(15) {
			t.Errorf("esperava descansoMinutos 15, got: %v", resp["descansoMinutos"])
		}
	})

	t.Run("retorna 400 quando descanso é negativo", func(t *testing.T) {
		r, _ := novoRouterPreferencias(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body, _ := json.Marshal(map[string]any{"telefone": "11999998888", "aceitaAgendamentos": true, "descansoMinutos": -5, "duracaoAtendimentoMinutos": 60})
		req := httptest.NewRequest(http.MethodPut, "/providers/me/preferencias", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("retorna 403 quando cliente tenta atualizar preferências de prestador", func(t *testing.T) {
		r, _ := novoRouterPreferencias(t)
		cookie := loginEObterCookie(t, r, "/auth/client/login", "maria@email.com", "12345678")

		body, _ := json.Marshal(map[string]any{"telefone": "11999998888", "aceitaAgendamentos": true, "descansoMinutos": 10, "duracaoAtendimentoMinutos": 60})
		req := httptest.NewRequest(http.MethodPut, "/providers/me/preferencias", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("esperava 403, got: %d", rr.Code)
		}
	})

	t.Run("retorna 401 sem cookie de sessão", func(t *testing.T) {
		r, _ := novoRouterPreferencias(t)

		body, _ := json.Marshal(map[string]any{"telefone": "11999998888", "aceitaAgendamentos": true, "descansoMinutos": 10, "duracaoAtendimentoMinutos": 60})
		req := httptest.NewRequest(http.MethodPut, "/providers/me/preferencias", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("esperava 401, got: %d", rr.Code)
		}
	})

	t.Run("persiste três blocos curtos como expediente padrão", func(t *testing.T) {
		r, _ := novoRouterPreferencias(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body, _ := json.Marshal(map[string]any{
			"telefone":                  "11999998888",
			"aceitaAgendamentos":        true,
			"descansoMinutos":           10,
			"duracaoAtendimentoMinutos": 60,
			"horariosPadrao": []map[string]int{
				{"inicioMinutos": 8 * 60, "fimMinutos": 10 * 60},
				{"inicioMinutos": 11 * 60, "fimMinutos": 13 * 60},
				{"inicioMinutos": 15 * 60, "fimMinutos": 17 * 60},
			},
		})
		req := httptest.NewRequest(http.MethodPut, "/providers/me/preferencias", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d, body: %s", rr.Code, rr.Body.String())
		}
		var resp map[string]any
		json.NewDecoder(rr.Body).Decode(&resp)
		horarios := resp["horariosPadrao"].([]any)
		if len(horarios) != 3 {
			t.Errorf("esperava 3 blocos, got: %v", horarios)
		}
	})

	t.Run("retorna 400 quando um bloco do expediente padrão é inválido", func(t *testing.T) {
		r, _ := novoRouterPreferencias(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		body, _ := json.Marshal(map[string]any{
			"telefone":                  "11999998888",
			"aceitaAgendamentos":        true,
			"descansoMinutos":           0,
			"duracaoAtendimentoMinutos": 60,
			"horariosPadrao":            []map[string]int{{"inicioMinutos": 605, "fimMinutos": 720}},
		})
		req := httptest.NewRequest(http.MethodPut, "/providers/me/preferencias", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})
}
