package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agendago/internal/adapter/email"
	"agendago/internal/adapter/http/handler"
	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	"agendago/internal/domain/client"
	"agendago/internal/domain/precadastro"
	"agendago/internal/pkg/token"
	ucclient "agendago/internal/usecase/client"

	"github.com/go-chi/chi/v5"
)

func novoClientHandler() (*handler.ClientHandler, *repository.ClientMemoria, *repository.PreCadastroMemoria) {
	clients := repository.NovoClientMemoria()
	providers := repository.NovoProviderMemoria()
	pendentes := repository.NovoSignupMemoria()
	notificador := email.NovoNotificador(email.NovaMailerMemoria(), "http://localhost:5173", time.UTC, email.ExecutorSincrono)
	hasher := security.NovoHasherArgon2id()
	solicitar := ucclient.NovoSolicitarCadastroUseCase(clients, providers, pendentes, notificador, hasher)
	confirmar := ucclient.NovoConfirmarCadastroUseCase(clients, providers, pendentes)
	preCadastroRepo := repository.NovoPreCadastroMemoria()
	consultarPreCadastro := ucclient.NovoConsultarPreCadastroUseCase(preCadastroRepo)
	concluirPreCadastro := ucclient.NovoConcluirPreCadastroUseCase(clients, providers, preCadastroRepo, hasher)
	return handler.NovoClientHandler(solicitar, confirmar, consultarPreCadastro, concluirPreCadastro), clients, preCadastroRepo
}

// requisicaoComToken monta uma requisição com o token como URL param do chi,
// simulando o roteamento real para handlers que usam chi.URLParam.
func requisicaoComToken(method, rota, token string) *http.Request {
	req := httptest.NewRequest(method, rota, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("token", token)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// requisicaoComTokenEBody é requisicaoComToken com um corpo JSON, para
// handlers que recebem token na URL e dados no body (ex: ConcluirPreCadastro).
func requisicaoComTokenEBody(method, rota, token string, body any) *http.Request {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(method, rota, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("token", token)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func fazerRequisicaoClient(t *testing.T, h *handler.ClientHandler, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Cadastrar(rr, req)
	return rr
}

func corpoCadastro() map[string]string {
	return map[string]string{
		"nome":     "Maria Silva",
		"email":    "maria@email.com",
		"telefone": "11999998888",
		"senha":    "12345678",
	}
}

func TestHandlerCadastrarClient(t *testing.T) {
	t.Run("retorna 204 para dados válidos (email de confirmação enviado)", func(t *testing.T) {
		h, _, _ := novoClientHandler()
		rr := fazerRequisicaoClient(t, h, corpoCadastro())
		if rr.Code != http.StatusNoContent {
			t.Errorf("esperava 204, got: %d, body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("retorna o mesmo 204 para email já cadastrado (anti-enumeração)", func(t *testing.T) {
		h, clients, _ := novoClientHandler()
		conta, _ := client.NovoComConta("c-1", "Maria", "maria@email.com", "hash-existente")
		clients.Salvar(conta)

		rr := fazerRequisicaoClient(t, h, corpoCadastro())
		if rr.Code != http.StatusNoContent {
			t.Errorf("esperava 204 (mesma resposta), got: %d", rr.Code)
		}
	})

	t.Run("retorna 400 quando email não tem formato válido", func(t *testing.T) {
		h, _, _ := novoClientHandler()
		corpo := corpoCadastro()
		corpo["email"] = "emailinvalido"
		rr := fazerRequisicaoClient(t, h, corpo)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("retorna 400 quando senha tem menos de 8 caracteres", func(t *testing.T) {
		h, _, _ := novoClientHandler()
		corpo := corpoCadastro()
		corpo["senha"] = "123"
		rr := fazerRequisicaoClient(t, h, corpo)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("retorna 400 quando telefone é curto demais", func(t *testing.T) {
		h, _, _ := novoClientHandler()
		corpo := corpoCadastro()
		corpo["telefone"] = "123"
		rr := fazerRequisicaoClient(t, h, corpo)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("retorna 400 quando body não é um JSON válido", func(t *testing.T) {
		h, _, _ := novoClientHandler()
		req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewBufferString("não é json"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.Cadastrar(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})
}

func TestHandlerConfirmarCadastro(t *testing.T) {
	t.Run("retorna 400 para token inválido", func(t *testing.T) {
		h, _, _ := novoClientHandler()
		b, _ := json.Marshal(map[string]string{"token": "token-inexistente"})
		req := httptest.NewRequest(http.MethodPost, "/clients/confirmar-cadastro", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.ConfirmarCadastro(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("retorna 400 quando falta o token", func(t *testing.T) {
		h, _, _ := novoClientHandler()
		b, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest(http.MethodPost, "/clients/confirmar-cadastro", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.ConfirmarCadastro(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})
}

func TestHandlerConsultarPreCadastro(t *testing.T) {
	t.Run("retorna os dados do convidado para um token válido", func(t *testing.T) {
		h, _, preCadastros := novoClientHandler()
		preCadastros.Salvar(precadastro.Novo(token.Hash("token-valido"), "Convidada Silva", "convidada@email.com", "11999998888"))

		req := requisicaoComToken(http.MethodGet, "/clients/pre-cadastro/token-valido", "token-valido")
		rr := httptest.NewRecorder()
		h.ConsultarPreCadastro(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d, body: %s", rr.Code, rr.Body.String())
		}

		var out map[string]string
		json.NewDecoder(rr.Body).Decode(&out)
		if out["nome"] != "Convidada Silva" || out["email"] != "convidada@email.com" || out["telefone"] != "11999998888" {
			t.Errorf("dados inesperados: %+v", out)
		}
	})

	t.Run("consultar o mesmo token mais de uma vez continua funcionando (não consome)", func(t *testing.T) {
		h, _, preCadastros := novoClientHandler()
		preCadastros.Salvar(precadastro.Novo(token.Hash("token-repetido"), "Convidada", "convidada@email.com", "11999998888"))

		req1 := requisicaoComToken(http.MethodGet, "/clients/pre-cadastro/token-repetido", "token-repetido")
		rr1 := httptest.NewRecorder()
		h.ConsultarPreCadastro(rr1, req1)
		if rr1.Code != http.StatusOK {
			t.Fatalf("esperava 200 na primeira consulta, got: %d", rr1.Code)
		}

		req2 := requisicaoComToken(http.MethodGet, "/clients/pre-cadastro/token-repetido", "token-repetido")
		rr2 := httptest.NewRecorder()
		h.ConsultarPreCadastro(rr2, req2)
		if rr2.Code != http.StatusOK {
			t.Errorf("esperava 200 também na segunda consulta (só o submit final consome), got: %d", rr2.Code)
		}
	})

	t.Run("retorna 400 para token inexistente", func(t *testing.T) {
		h, _, _ := novoClientHandler()
		req := requisicaoComToken(http.MethodGet, "/clients/pre-cadastro/token-inexistente", "token-inexistente")
		rr := httptest.NewRecorder()
		h.ConsultarPreCadastro(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})
}

func TestHandlerConcluirPreCadastro(t *testing.T) {
	t.Run("cria a conta direto, sem segunda confirmação por email", func(t *testing.T) {
		h, clients, preCadastros := novoClientHandler()
		preCadastros.Salvar(precadastro.Novo(token.Hash("token-conclusao"), "Convidada Silva", "convidada@email.com", "11999998888"))

		req := requisicaoComTokenEBody(http.MethodPost, "/clients/pre-cadastro/token-conclusao", "token-conclusao", map[string]string{"senha": "SenhaForte123"})
		rr := httptest.NewRecorder()
		h.ConcluirPreCadastro(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d, body: %s", rr.Code, rr.Body.String())
		}

		var out map[string]string
		json.NewDecoder(rr.Body).Decode(&out)
		if out["email"] != "convidada@email.com" || out["nome"] != "Convidada Silva" {
			t.Errorf("dados inesperados: %+v", out)
		}

		conta, _ := clients.BuscarPorEmail("convidada@email.com")
		if conta == nil || !conta.TemConta() {
			t.Error("esperava conta criada com senha definida")
		}
	})

	t.Run("token de pré-cadastro é consumido: segunda conclusão falha", func(t *testing.T) {
		h, _, preCadastros := novoClientHandler()
		preCadastros.Salvar(precadastro.Novo(token.Hash("token-unico"), "Convidada", "convidada2@email.com", "11999998888"))

		corpo := map[string]string{"senha": "SenhaForte123"}
		req1 := requisicaoComTokenEBody(http.MethodPost, "/clients/pre-cadastro/token-unico", "token-unico", corpo)
		h.ConcluirPreCadastro(httptest.NewRecorder(), req1)

		req2 := requisicaoComTokenEBody(http.MethodPost, "/clients/pre-cadastro/token-unico", "token-unico", corpo)
		rr2 := httptest.NewRecorder()
		h.ConcluirPreCadastro(rr2, req2)
		if rr2.Code != http.StatusBadRequest {
			t.Errorf("esperava 400 na segunda conclusão, got: %d", rr2.Code)
		}
	})

	t.Run("retorna 400 para token inexistente", func(t *testing.T) {
		h, _, _ := novoClientHandler()
		req := requisicaoComTokenEBody(http.MethodPost, "/clients/pre-cadastro/token-inexistente", "token-inexistente", map[string]string{"senha": "SenhaForte123"})
		rr := httptest.NewRecorder()
		h.ConcluirPreCadastro(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("retorna 400 para senha curta", func(t *testing.T) {
		h, _, preCadastros := novoClientHandler()
		preCadastros.Salvar(precadastro.Novo(token.Hash("token-senha-curta"), "Convidada", "convidada3@email.com", "11999998888"))

		req := requisicaoComTokenEBody(http.MethodPost, "/clients/pre-cadastro/token-senha-curta", "token-senha-curta", map[string]string{"senha": "123"})
		rr := httptest.NewRecorder()
		h.ConcluirPreCadastro(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})
}
