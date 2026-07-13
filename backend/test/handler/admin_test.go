package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"agendago/internal/adapter/http/handler"
	"agendago/internal/adapter/http/middleware"
	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
	ucadmin "agendago/internal/usecase/admin"
	ucappointment "agendago/internal/usecase/appointment"
	ucauth "agendago/internal/usecase/auth"

	"github.com/go-chi/chi/v5"
)

// novoRouterAdmin monta um router com um admin semeado, um prestador e um
// cliente, e as rotas de moderação sob ExigirAdmin — espelhando main.go.
func novoRouterAdmin(t *testing.T) (r *chi.Mux, providerID, clientID string) {
	t.Helper()
	hasher := security.NovoHasherArgon2id()

	providerRepo := repository.NovoProviderMemoria()
	clientRepo := repository.NovoClientMemoria()
	sessionRepo := repository.NovoSessionMemoria()
	adminRepo := repository.NovoAdminMemoria()

	if err := ucadmin.NovoSemearUseCase(adminRepo, hasher).Executar("admin@agendago.dev", "12345678"); err != nil {
		t.Fatalf("semear admin: %v", err)
	}

	senhaHash, _ := hasher.Gerar("12345678")
	p, _ := provider.Novo("11111111-1111-1111-1111-111111111111", "João Silva", "joao@email.com", "11999998888", senhaHash)
	p.AtivarAgenda()
	providerRepo.Salvar(p)
	c, _ := client.NovoComConta("22222222-2222-2222-2222-222222222222", "Maria Souza", "maria@email.com", senhaHash)
	clientRepo.Salvar(c)

	loginProvider := ucauth.NovoLoginProviderUseCase(providerRepo, sessionRepo, hasher)
	loginClient := ucauth.NovoLoginClientUseCase(clientRepo, sessionRepo, hasher)
	loginAdmin := ucauth.NovoLoginAdminUseCase(adminRepo, sessionRepo, hasher)
	validarSessao := ucauth.NovoValidarSessaoUseCase(sessionRepo)

	identidadeDoContexto := func(req *http.Request) (ucauth.Identidade, bool) {
		return middleware.IdentidadeDoContexto(req.Context())
	}
	moderar := ucadmin.NovoModerarUseCase(providerRepo, clientRepo, sessionRepo)
	appointmentRepo := repository.NovoAppointmentMemoria()
	listarAgendamentos := ucappointment.NovoListarUseCase(appointmentRepo, providerRepo, clientRepo)
	detalhar := ucadmin.NovoDetalharUseCase(providerRepo, clientRepo, listarAgendamentos)
	adminHandler := handler.NovoAdminHandler(moderar, detalhar)
	authHandler := handler.NovoAuthHandler(loginProvider, loginClient, loginAdmin, nil, nil, false, identidadeDoContexto)
	authMw := middleware.NovoAuth(validarSessao)

	router := chi.NewRouter()
	router.Post("/auth/provider/login", authHandler.LoginProvider)
	router.Post("/auth/admin/login", authHandler.LoginAdmin)
	router.Group(func(router chi.Router) {
		router.Use(authMw.Autenticar)
		router.Use(middleware.ExigirAdmin)
		router.Get("/admin/prestadores", adminHandler.ListarPrestadores)
		router.Get("/admin/prestadores/{id}", adminHandler.DetalharPrestador)
		router.Get("/admin/clientes", adminHandler.ListarClientes)
		router.Get("/admin/clientes/{id}", adminHandler.DetalharCliente)
		router.Post("/admin/prestadores/{id}/banir", adminHandler.BanirPrestador)
		router.Post("/admin/prestadores/{id}/reativar", adminHandler.ReativarPrestador)
		router.Post("/admin/clientes/{id}/banir", adminHandler.BanirCliente)
		router.Post("/admin/clientes/{id}/reativar", adminHandler.ReativarCliente)
	})

	return router, p.ID, c.ID
}

func TestHandlerAdmin(t *testing.T) {
	t.Run("admin lista, bane um prestador e o prestador banido não loga mais", func(t *testing.T) {
		r, providerID, _ := novoRouterAdmin(t)
		cookieAdmin := loginEObterCookie(t, r, "/auth/admin/login", "admin@agendago.dev", "12345678")

		rr := requisicaoComCookie(t, r, http.MethodGet, "/admin/prestadores", nil, cookieAdmin)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d", rr.Code)
		}
		var lista map[string]any
		json.NewDecoder(rr.Body).Decode(&lista)
		if len(lista["usuarios"].([]any)) != 1 {
			t.Fatalf("esperava 1 prestador, got: %v", lista["usuarios"])
		}

		// banir
		rr = requisicaoComCookie(t, r, http.MethodPost, "/admin/prestadores/"+providerID+"/banir", nil, cookieAdmin)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("esperava 204 no banimento, got: %d, body: %s", rr.Code, rr.Body.String())
		}

		// prestador banido: login bloqueado (403)
		rr = requisicaoComCookie(t, r, http.MethodPost, "/auth/provider/login",
			map[string]string{"email": "joao@email.com", "senha": "12345678"}, nil)
		if rr.Code != http.StatusForbidden {
			t.Errorf("esperava 403 no login do banido, got: %d", rr.Code)
		}

		// reativar devolve o acesso
		rr = requisicaoComCookie(t, r, http.MethodPost, "/admin/prestadores/"+providerID+"/reativar", nil, cookieAdmin)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("esperava 204 na reativação, got: %d", rr.Code)
		}
		rr = requisicaoComCookie(t, r, http.MethodPost, "/auth/provider/login",
			map[string]string{"email": "joao@email.com", "senha": "12345678"}, nil)
		if rr.Code != http.StatusOK {
			t.Errorf("esperava 200 no login após reativar, got: %d", rr.Code)
		}
	})

	t.Run("banir cliente com sucesso e id inexistente retorna 404", func(t *testing.T) {
		r, _, clientID := novoRouterAdmin(t)
		cookieAdmin := loginEObterCookie(t, r, "/auth/admin/login", "admin@agendago.dev", "12345678")

		rr := requisicaoComCookie(t, r, http.MethodPost, "/admin/clientes/"+clientID+"/banir", nil, cookieAdmin)
		if rr.Code != http.StatusNoContent {
			t.Errorf("esperava 204, got: %d", rr.Code)
		}
		rr = requisicaoComCookie(t, r, http.MethodPost, "/admin/clientes/fantasma/banir", nil, cookieAdmin)
		if rr.Code != http.StatusNotFound {
			t.Errorf("esperava 404, got: %d", rr.Code)
		}
	})

	t.Run("lista clientes com o status de moderação e reativa um banido", func(t *testing.T) {
		r, _, clientID := novoRouterAdmin(t)
		cookieAdmin := loginEObterCookie(t, r, "/auth/admin/login", "admin@agendago.dev", "12345678")

		requisicaoComCookie(t, r, http.MethodPost, "/admin/clientes/"+clientID+"/banir", nil, cookieAdmin)

		rr := requisicaoComCookie(t, r, http.MethodGet, "/admin/clientes", nil, cookieAdmin)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200 na listagem de clientes, got: %d", rr.Code)
		}
		var lista map[string]any
		json.NewDecoder(rr.Body).Decode(&lista)
		usuarios := lista["usuarios"].([]any)
		if len(usuarios) != 1 {
			t.Fatalf("esperava 1 cliente na moderação, got: %d", len(usuarios))
		}
		if usuarios[0].(map[string]any)["ativo"] != false {
			t.Error("esperava cliente banido na listagem")
		}

		rr = requisicaoComCookie(t, r, http.MethodPost, "/admin/clientes/"+clientID+"/reativar", nil, cookieAdmin)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("esperava 204 na reativação, got: %d", rr.Code)
		}
		rr = requisicaoComCookie(t, r, http.MethodGet, "/admin/clientes", nil, cookieAdmin)
		json.NewDecoder(rr.Body).Decode(&lista)
		if lista["usuarios"].([]any)[0].(map[string]any)["ativo"] != true {
			t.Error("esperava cliente ativo após reativar")
		}
	})

	t.Run("admin detalha prestador e cliente; id inexistente retorna 404", func(t *testing.T) {
		r, providerID, clientID := novoRouterAdmin(t)
		cookieAdmin := loginEObterCookie(t, r, "/auth/admin/login", "admin@agendago.dev", "12345678")

		rr := requisicaoComCookie(t, r, http.MethodGet, "/admin/prestadores/"+providerID, nil, cookieAdmin)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200 no detalhe do prestador, got: %d, body: %s", rr.Code, rr.Body.String())
		}
		var dp map[string]any
		json.NewDecoder(rr.Body).Decode(&dp)
		if dp["nome"] != "João Silva" || dp["ativo"] != true {
			t.Errorf("esperava dados cadastrais do prestador, got: %+v", dp)
		}
		if _, ok := dp["agendamentos"].([]any); !ok {
			t.Errorf("esperava a lista de agendamentos no detalhe, got: %+v", dp["agendamentos"])
		}

		rr = requisicaoComCookie(t, r, http.MethodGet, "/admin/clientes/"+clientID, nil, cookieAdmin)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200 no detalhe do cliente, got: %d, body: %s", rr.Code, rr.Body.String())
		}
		var dc map[string]any
		json.NewDecoder(rr.Body).Decode(&dc)
		if dc["nome"] != "Maria Souza" || dc["temConta"] != true {
			t.Errorf("esperava dados cadastrais do cliente, got: %+v", dc)
		}

		rr = requisicaoComCookie(t, r, http.MethodGet, "/admin/prestadores/fantasma", nil, cookieAdmin)
		if rr.Code != http.StatusNotFound {
			t.Errorf("esperava 404 para prestador inexistente, got: %d", rr.Code)
		}
		rr = requisicaoComCookie(t, r, http.MethodGet, "/admin/clientes/fantasma", nil, cookieAdmin)
		if rr.Code != http.StatusNotFound {
			t.Errorf("esperava 404 para cliente inexistente, got: %d", rr.Code)
		}
	})

	t.Run("prestador não acessa rotas de admin (403) e sem sessão (401)", func(t *testing.T) {
		r, _, _ := novoRouterAdmin(t)

		rrSemSessao := requisicaoComCookie(t, r, http.MethodGet, "/admin/prestadores", nil, nil)
		if rrSemSessao.Code != http.StatusUnauthorized {
			t.Errorf("esperava 401 sem sessão, got: %d", rrSemSessao.Code)
		}

		cookiePrestador := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")
		rrPrestador := requisicaoComCookie(t, r, http.MethodGet, "/admin/prestadores", nil, cookiePrestador)
		if rrPrestador.Code != http.StatusForbidden {
			t.Errorf("esperava 403 para prestador, got: %d", rrPrestador.Code)
		}
	})
}
