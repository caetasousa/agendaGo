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
	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	"agendago/internal/domain/provider"
	ucauth "agendago/internal/usecase/auth"

	"github.com/go-chi/chi/v5"
)

// novoRouterPasswordReset monta um router chi só com as rotas de recuperação
// de senha, com um prestador já cadastrado, espelhando o wiring de main.go.
func novoRouterPasswordReset(t *testing.T) (r *chi.Mux, mailer *email.MailerMemoria) {
	t.Helper()
	hasher := security.NovoHasherArgon2id()

	providerRepo := repository.NovoProviderMemoria()
	clientRepo := repository.NovoClientMemoria()
	resetRepo := repository.NovoPasswordResetMemoria()
	sessionRepo := repository.NovoSessionMemoria()

	senhaHash, _ := hasher.Gerar("12345678")
	p, _ := provider.Novo("provider-1", "João Silva", "joao@email.com", "11999998888", senhaHash)
	providerRepo.Salvar(p)

	mailer = email.NovaMailerMemoria()
	notificador := email.NovoNotificador(mailer, "http://localhost:5173", time.UTC, email.ExecutorSincrono)

	solicitar := ucauth.NovoSolicitarRecuperacaoUseCase(providerRepo, clientRepo, resetRepo, notificador)
	redefinir := ucauth.NovoRedefinirSenhaUseCase(providerRepo, clientRepo, resetRepo, sessionRepo, hasher)
	passwordResetHandler := handler.NovoPasswordResetHandler(solicitar, redefinir)

	router := chi.NewRouter()
	router.Post("/auth/recuperar-senha", passwordResetHandler.Solicitar)
	router.Post("/auth/redefinir-senha", passwordResetHandler.Redefinir)

	return router, mailer
}

func TestSolicitarRecuperacaoHandler(t *testing.T) {
	t.Run("retorna 204 para email existente e envia o email", func(t *testing.T) {
		r, mailer := novoRouterPasswordReset(t)
		body, _ := json.Marshal(map[string]string{"email": "joao@email.com"})
		req := httptest.NewRequest(http.MethodPost, "/auth/recuperar-senha", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("esperava 204, got: %d", rr.Code)
		}
		if len(mailer.Enviadas()) != 1 {
			t.Errorf("esperava 1 email enviado, got: %d", len(mailer.Enviadas()))
		}
	})

	t.Run("retorna o mesmo 204 para email inexistente, sem enviar email", func(t *testing.T) {
		r, mailer := novoRouterPasswordReset(t)
		body, _ := json.Marshal(map[string]string{"email": "fantasma@email.com"})
		req := httptest.NewRequest(http.MethodPost, "/auth/recuperar-senha", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("esperava 204 (mesma resposta de email existente), got: %d", rr.Code)
		}
		if len(mailer.Enviadas()) != 0 {
			t.Errorf("esperava zero emails, got: %d", len(mailer.Enviadas()))
		}
	})

	t.Run("retorna 400 para email malformado", func(t *testing.T) {
		r, _ := novoRouterPasswordReset(t)
		body, _ := json.Marshal(map[string]string{"email": "não-é-um-email"})
		req := httptest.NewRequest(http.MethodPost, "/auth/recuperar-senha", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})
}

func TestRedefinirSenhaHandler(t *testing.T) {
	t.Run("retorna 400 para token inválido", func(t *testing.T) {
		r, _ := novoRouterPasswordReset(t)
		body, _ := json.Marshal(map[string]string{"token": "token-inexistente", "novaSenha": "senha-nova-123"})
		req := httptest.NewRequest(http.MethodPost, "/auth/redefinir-senha", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, got: %d", rr.Code)
		}
		var resp map[string]string
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["erro"] != ucauth.ErrTokenRecuperacaoInvalido.Error() {
			t.Errorf("esperava erro %q, got: %v", ucauth.ErrTokenRecuperacaoInvalido.Error(), resp)
		}
	})

	t.Run("retorna 400 para senha curta demais", func(t *testing.T) {
		r, _ := novoRouterPasswordReset(t)
		body, _ := json.Marshal(map[string]string{"token": "qualquer-coisa", "novaSenha": "curta"})
		req := httptest.NewRequest(http.MethodPost, "/auth/redefinir-senha", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("fluxo completo: solicitar e redefinir com sucesso", func(t *testing.T) {
		r, mailer := novoRouterPasswordReset(t)

		solicitarBody, _ := json.Marshal(map[string]string{"email": "joao@email.com"})
		solicitarReq := httptest.NewRequest(http.MethodPost, "/auth/recuperar-senha", bytes.NewReader(solicitarBody))
		solicitarReq.Header.Set("Content-Type", "application/json")
		solicitarRR := httptest.NewRecorder()
		r.ServeHTTP(solicitarRR, solicitarReq)
		if solicitarRR.Code != http.StatusNoContent {
			t.Fatalf("solicitação de base falhou: %d", solicitarRR.Code)
		}

		enviadas := mailer.Enviadas()
		if len(enviadas) != 1 {
			t.Fatalf("esperava 1 email, got: %d", len(enviadas))
		}
		token := tokenDoLink(t, enviadas[len(enviadas)-1].HTML)

		redefinirBody, _ := json.Marshal(map[string]string{"token": token, "novaSenha": "senha-nova-123"})
		redefinirReq := httptest.NewRequest(http.MethodPost, "/auth/redefinir-senha", bytes.NewReader(redefinirBody))
		redefinirReq.Header.Set("Content-Type", "application/json")
		redefinirRR := httptest.NewRecorder()
		r.ServeHTTP(redefinirRR, redefinirReq)

		if redefinirRR.Code != http.StatusNoContent {
			t.Fatalf("esperava 204, got: %d", redefinirRR.Code)
		}
	})
}

// tokenDoLink extrai o token do link presente no HTML de um email de
// recuperação de senha, no formato `.../redefinir-senha?token=XXX`.
func tokenDoLink(t *testing.T, html string) string {
	t.Helper()
	const marcador = "token="
	i := bytes.Index([]byte(html), []byte(marcador))
	if i < 0 {
		t.Fatalf("email sem link de token: %s", html)
	}
	resto := html[i+len(marcador):]
	fim := bytes.IndexByte([]byte(resto), '"')
	if fim < 0 {
		t.Fatalf("token sem fim de atributo: %s", html)
	}
	return resto[:fim]
}
