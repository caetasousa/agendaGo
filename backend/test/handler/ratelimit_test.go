package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agendago/internal/adapter/email"
	"agendago/internal/adapter/http/handler"
	"agendago/internal/adapter/http/middleware"
	"agendago/internal/adapter/security"
	ucauth "agendago/internal/usecase/auth"
	"agendago/test/repository/memoria"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

const (
	tetoPorIP    = 3
	tetoPorConta = 5
)

// novoRouterComTetos monta o login e a recuperação de senha com os DOIS tetos
// ligados, como em main.go: por IP no roteador e por conta dentro do handler.
// É essa combinação que o teste exercita — o teto por conta só importa quando
// o de IP é contornado.
func novoRouterComTetos(t *testing.T) *chi.Mux {
	t.Helper()
	hasher := security.NovoHasherArgon2id()

	usuarios, membros, providers := fakesDePrestador()
	clientRepo := memoria.NovoClientMemoria()
	sessionRepo := memoria.NovoSessionMemoria()
	resetRepo := memoria.NovoPasswordResetMemoria()

	senhaHash, _ := hasher.Gerar("12345678")
	_, p := criarPrestador(usuarios, membros, providers, "provider-1", "João Silva", "joao@email.com", "11999998888", senhaHash)
	providers.Salvar(p)

	notificador := email.NovoNotificador(email.NovaMailerMemoria(), "http://localhost:5173", time.UTC, email.ExecutorSincrono)

	limitador := handler.NovoLimitadorPorConta(tetoPorConta, time.Minute)
	authHandler := handler.NovoAuthHandler(
		ucauth.NovoLoginProviderUseCase(usuarios, membros, providers, sessionRepo, hasher),
		ucauth.NovoLoginClientUseCase(clientRepo, sessionRepo, hasher),
		nil, nil, nil, false, limitador, nil,
	)
	passwordResetHandler := handler.NovoPasswordResetHandler(
		ucauth.NovoSolicitarRecuperacaoUseCase(usuarios, membros, providers, clientRepo, resetRepo, notificador),
		ucauth.NovoRedefinirSenhaUseCase(usuarios, clientRepo, resetRepo, sessionRepo, hasher),
		limitador,
	)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitBy(tetoPorIP, time.Minute, middleware.ChavePorIP,
			httprate.WithLimitHandler(middleware.RespostaLimiteExcedido)))
		r.Post("/auth/provider/login", authHandler.LoginProvider)
		r.Post("/auth/recuperar-senha", passwordResetHandler.Solicitar)
	})
	return r
}

// postDeIP dispara a requisição como se viesse do IP informado — é assim que o
// teste simula o atacante que troca de endereço a cada tentativa.
func postDeIP(t *testing.T, r *chi.Mux, caminho, ip string, corpo map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(corpo)
	req := httptest.NewRequest(http.MethodPost, caminho, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":54321"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func ipDaTentativa(n int) string {
	return "203.0.113." + string(rune('0'+n))
}

func TestTetoPorConta(t *testing.T) {
	t.Run("bloqueia brute-force que troca de IP a cada tentativa", func(t *testing.T) {
		r := novoRouterComTetos(t)
		errada := map[string]string{"email": "joao@email.com", "senha": "senha-errada"}

		// cada tentativa vem de um IP novo: o teto por IP nunca é atingido
		for i := 0; i < tetoPorConta; i++ {
			rr := postDeIP(t, r, "/auth/provider/login", ipDaTentativa(i), errada)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("tentativa %d: esperava 401, got: %d", i+1, rr.Code)
			}
		}

		rr := postDeIP(t, r, "/auth/provider/login", ipDaTentativa(tetoPorConta), errada)
		if rr.Code != http.StatusTooManyRequests {
			t.Fatalf("esperava 429 depois de %d falhas na mesma conta, got: %d", tetoPorConta, rr.Code)
		}
		var resp map[string]string
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["erro"] == "" {
			t.Error("esperava corpo JSON com a chave erro")
		}
		if rr.Header().Get("Retry-After") == "" {
			t.Error("esperava cabeçalho Retry-After no 429")
		}
	})

	t.Run("login correto não conta para o teto", func(t *testing.T) {
		r := novoRouterComTetos(t)
		certa := map[string]string{"email": "joao@email.com", "senha": "12345678"}

		// mais logins bem-sucedidos que o teto: ninguém fica trancado fora da
		// própria conta por acertar a senha muitas vezes
		for i := 0; i < tetoPorConta+2; i++ {
			rr := postDeIP(t, r, "/auth/provider/login", ipDaTentativa(i), certa)
			if rr.Code != http.StatusOK {
				t.Fatalf("login %d: esperava 200, got: %d", i+1, rr.Code)
			}
		}
	})

	t.Run("email inexistente é bloqueado igual, sem revelar que não existe", func(t *testing.T) {
		r := novoRouterComTetos(t)
		fantasma := map[string]string{"email": "ninguem@email.com", "senha": "senha-errada"}

		for i := 0; i < tetoPorConta; i++ {
			if rr := postDeIP(t, r, "/auth/provider/login", ipDaTentativa(i), fantasma); rr.Code != http.StatusUnauthorized {
				t.Fatalf("tentativa %d: esperava 401, got: %d", i+1, rr.Code)
			}
		}

		if rr := postDeIP(t, r, "/auth/provider/login", ipDaTentativa(tetoPorConta), fantasma); rr.Code != http.StatusTooManyRequests {
			t.Fatalf("esperava 429 também para email inexistente, got: %d", rr.Code)
		}
	})

	t.Run("recuperação de senha limita o volume de emails para a mesma conta", func(t *testing.T) {
		r := novoRouterComTetos(t)
		pedido := map[string]string{"email": "joao@email.com"}

		for i := 0; i < tetoPorConta; i++ {
			if rr := postDeIP(t, r, "/auth/recuperar-senha", ipDaTentativa(i), pedido); rr.Code != http.StatusNoContent {
				t.Fatalf("pedido %d: esperava 204, got: %d", i+1, rr.Code)
			}
		}

		if rr := postDeIP(t, r, "/auth/recuperar-senha", ipDaTentativa(tetoPorConta), pedido); rr.Code != http.StatusTooManyRequests {
			t.Fatalf("esperava 429 depois de %d pedidos, got: %d", tetoPorConta, rr.Code)
		}
	})
}

func TestTetoPorIP(t *testing.T) {
	t.Run("bloqueia rajada do mesmo IP mesmo variando a conta", func(t *testing.T) {
		r := novoRouterComTetos(t)

		// contas diferentes a cada tentativa: o teto por conta nunca é atingido
		for i := 0; i < tetoPorIP; i++ {
			corpo := map[string]string{"email": "conta" + string(rune('a'+i)) + "@email.com", "senha": "x12345678"}
			if rr := postDeIP(t, r, "/auth/provider/login", "198.51.100.7", corpo); rr.Code != http.StatusUnauthorized {
				t.Fatalf("tentativa %d: esperava 401, got: %d", i+1, rr.Code)
			}
		}

		corpo := map[string]string{"email": "ultima@email.com", "senha": "x12345678"}
		if rr := postDeIP(t, r, "/auth/provider/login", "198.51.100.7", corpo); rr.Code != http.StatusTooManyRequests {
			t.Fatalf("esperava 429 depois de %d tentativas do mesmo IP, got: %d", tetoPorIP, rr.Code)
		}
	})
}
