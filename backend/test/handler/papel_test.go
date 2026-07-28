package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agendago/internal/adapter/http/handler"
	"agendago/internal/adapter/http/middleware"
	"agendago/internal/adapter/security"
	"agendago/internal/domain/membro"
	ucauth "agendago/internal/usecase/auth"
	ucprovider "agendago/internal/usecase/provider"
	"agendago/test/repository/memoria"

	"github.com/go-chi/chi/v5"
)

// Monta o grupo de rotas do prestador com o mesmo encadeamento de main.go —
// Autenticar → ExigirProvider → ExigirGestaoDaAgenda — e devolve o router junto
// do fake de vínculos, para o teste trocar o papel do usuário.
func novoRouterComPapel(t *testing.T, papel membro.Papel) *chi.Mux {
	t.Helper()
	hasher := security.NovoHasherArgon2id()
	senhaHash, _ := hasher.Gerar("12345678")

	usuarios, membros, providers := fakesDePrestador()
	sessionRepo := memoria.NovoSessionMemoria()

	criarPrestador(usuarios, membros, providers, "provider-1", "João Silva", "joao@email.com", "11999998888", senhaHash)

	// Sobrescreve o vínculo com o papel sob teste. Vai direto no struct porque
	// membro.Novo recusa papel inválido, e é justamente o papel desconhecido
	// que precisa ser exercitado aqui.
	membros.Salvar(&membro.Membro{
		ID:         "m-provider-1",
		UsuarioID:  "provider-1",
		ProviderID: "provider-1",
		Papel:      papel,
	})

	loginProvider := ucauth.NovoLoginProviderUseCase(usuarios, membros, providers, sessionRepo, hasher)
	validarSessao := ucauth.NovoValidarSessaoUseCase(sessionRepo, membros)
	identidadeDoContexto := func(req *http.Request) (ucauth.Identidade, bool) {
		return middleware.IdentidadeDoContexto(req.Context())
	}
	atualizarPreferencias := ucprovider.NovoAtualizarPreferenciasUseCase(providers, usuarios)
	providerHandler := handler.NovoProviderHandler(nil, nil, atualizarPreferencias, nil, nil, identidadeDoContexto)
	authHandler := handler.NovoAuthHandler(loginProvider, nil, nil, nil, nil, false, nil, identidadeDoContexto)
	authMw := middleware.NovoAuth(validarSessao, false)

	router := chi.NewRouter()
	router.Post("/auth/provider/login", authHandler.LoginProvider)
	router.Group(func(router chi.Router) {
		router.Use(authMw.Autenticar)
		router.Use(middleware.ExigirProvider)
		router.Use(middleware.ExigirGestaoDaAgenda)
		router.Put("/providers/me/preferencias", providerHandler.AtualizarPreferencias)
	})
	return router
}

func preferenciasValidas() []byte {
	body, _ := json.Marshal(map[string]any{
		"telefone":                     "11999998888",
		"aceitaAgendamentos":           true,
		"descansoMinutos":              0,
		"duracaoAtendimentoMinutos":    60,
		"horariosPadrao":               []map[string]int{{"inicioMinutos": 480, "fimMinutos": 720}},
		"permiteMarcacaoPeloPrestador": true,
	})
	return body
}

func TestExigirGestaoDaAgenda(t *testing.T) {
	chamar := func(t *testing.T, papel membro.Papel) int {
		t.Helper()
		r := novoRouterComPapel(t, papel)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		req := httptest.NewRequest(http.MethodPut, "/providers/me/preferencias", bytes.NewReader(preferenciasValidas()))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr.Code
	}

	t.Run("dono opera a agenda", func(t *testing.T) {
		if code := chamar(t, membro.PapelDono); code != http.StatusOK {
			t.Errorf("esperava 200 para o dono, got: %d", code)
		}
	})

	t.Run("operador também opera a agenda", func(t *testing.T) {
		if code := chamar(t, membro.PapelOperador); code != http.StatusOK {
			t.Errorf("esperava 200 para o operador, got: %d", code)
		}
	})

	// O caso que justifica o middleware existir: sem ele, um papel que o
	// domínio não reconhece entraria em todas as rotas da agenda.
	t.Run("papel desconhecido é barrado com 403", func(t *testing.T) {
		if code := chamar(t, membro.Papel("leitor")); code != http.StatusForbidden {
			t.Errorf("esperava 403 para papel desconhecido, got: %d", code)
		}
	})
}

func TestExigirAdministracaoDaConta(t *testing.T) {
	// Rota fictícia: nenhuma rota de produção exige o dono ainda, mas o
	// middleware precisa estar correto antes de a primeira aparecer.
	montar := func(t *testing.T, papel membro.Papel) (*chi.Mux, *http.Cookie) {
		t.Helper()
		hasher := security.NovoHasherArgon2id()
		senhaHash, _ := hasher.Gerar("12345678")

		usuarios, membros, providers := fakesDePrestador()
		sessionRepo := memoria.NovoSessionMemoria()
		criarPrestador(usuarios, membros, providers, "provider-1", "João Silva", "joao@email.com", "11999998888", senhaHash)
		membros.Salvar(&membro.Membro{
			ID: "m-provider-1", UsuarioID: "provider-1", ProviderID: "provider-1", Papel: papel,
		})

		loginProvider := ucauth.NovoLoginProviderUseCase(usuarios, membros, providers, sessionRepo, hasher)
		validarSessao := ucauth.NovoValidarSessaoUseCase(sessionRepo, membros)
		identidadeDoContexto := func(req *http.Request) (ucauth.Identidade, bool) {
			return middleware.IdentidadeDoContexto(req.Context())
		}
		authHandler := handler.NovoAuthHandler(loginProvider, nil, nil, nil, nil, false, nil, identidadeDoContexto)
		authMw := middleware.NovoAuth(validarSessao, false)

		router := chi.NewRouter()
		router.Post("/auth/provider/login", authHandler.LoginProvider)
		router.Group(func(router chi.Router) {
			router.Use(authMw.Autenticar)
			router.Use(middleware.ExigirProvider)
			router.Use(middleware.ExigirAdministracaoDaConta)
			router.Delete("/providers/me/conta", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})
		})
		return router, loginEObterCookie(t, router, "/auth/provider/login", "joao@email.com", "12345678")
	}

	chamar := func(t *testing.T, papel membro.Papel) int {
		t.Helper()
		r, cookie := montar(t, papel)
		req := httptest.NewRequest(http.MethodDelete, "/providers/me/conta", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr.Code
	}

	t.Run("dono administra a conta", func(t *testing.T) {
		if code := chamar(t, membro.PapelDono); code != http.StatusNoContent {
			t.Errorf("esperava 204 para o dono, got: %d", code)
		}
	})

	t.Run("operador não administra a conta", func(t *testing.T) {
		if code := chamar(t, membro.PapelOperador); code != http.StatusForbidden {
			t.Errorf("esperava 403 para o operador, got: %d", code)
		}
	})
}
