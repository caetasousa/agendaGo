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
	"agendago/internal/domain/client"
	ucclient "agendago/internal/usecase/client"
)

func novoClientHandler() (*handler.ClientHandler, *repository.ClientMemoria) {
	clients := repository.NovoClientMemoria()
	providers := repository.NovoProviderMemoria()
	pendentes := repository.NovoSignupMemoria()
	notificador := email.NovoNotificador(email.NovaMailerMemoria(), "http://localhost:5173", time.UTC, email.ExecutorSincrono)
	hasher := security.NovoHasherArgon2id()
	solicitar := ucclient.NovoSolicitarCadastroUseCase(clients, providers, pendentes, notificador, hasher)
	confirmar := ucclient.NovoConfirmarCadastroUseCase(clients, providers, pendentes)
	return handler.NovoClientHandler(solicitar, confirmar), clients
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
		h, _ := novoClientHandler()
		rr := fazerRequisicaoClient(t, h, corpoCadastro())
		if rr.Code != http.StatusNoContent {
			t.Errorf("esperava 204, got: %d, body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("retorna o mesmo 204 para email já cadastrado (anti-enumeração)", func(t *testing.T) {
		h, clients := novoClientHandler()
		conta, _ := client.NovoComConta("c-1", "Maria", "maria@email.com", "hash-existente")
		clients.Salvar(conta)

		rr := fazerRequisicaoClient(t, h, corpoCadastro())
		if rr.Code != http.StatusNoContent {
			t.Errorf("esperava 204 (mesma resposta), got: %d", rr.Code)
		}
	})

	t.Run("retorna 400 quando email não tem formato válido", func(t *testing.T) {
		h, _ := novoClientHandler()
		corpo := corpoCadastro()
		corpo["email"] = "emailinvalido"
		rr := fazerRequisicaoClient(t, h, corpo)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("retorna 400 quando senha tem menos de 8 caracteres", func(t *testing.T) {
		h, _ := novoClientHandler()
		corpo := corpoCadastro()
		corpo["senha"] = "123"
		rr := fazerRequisicaoClient(t, h, corpo)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("retorna 400 quando telefone é curto demais", func(t *testing.T) {
		h, _ := novoClientHandler()
		corpo := corpoCadastro()
		corpo["telefone"] = "123"
		rr := fazerRequisicaoClient(t, h, corpo)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("retorna 400 quando body não é um JSON válido", func(t *testing.T) {
		h, _ := novoClientHandler()
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
		h, _ := novoClientHandler()
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
		h, _ := novoClientHandler()
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
