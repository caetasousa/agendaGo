package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agendago/internal/adapter/http/handler"
	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	ucprovider "agendago/internal/usecase/provider"
)

func novoHandler() *handler.ProviderHandler {
	repo := repository.NovoProviderMemoria()
	uc := ucprovider.NovoCadastrarUseCase(repo, security.NovoHasherArgon2id())
	return handler.NovoProviderHandler(uc)
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

func TestHandlerCadastrarProvider(t *testing.T) {
	t.Run("retorna 201 e ID do provider criado quando dados são válidos", func(t *testing.T) {
		rr := fazerRequisicao(t, novoHandler(), map[string]string{
			"nome":  "João Silva",
			"email": "joao@email.com",
			"senha": "12345678",
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
			"nome":  "João Silva",
			"email": "emailinvalido",
			"senha": "12345678",
		})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("retorna 400 quando senha tem menos de 8 caracteres", func(t *testing.T) {
		rr := fazerRequisicao(t, novoHandler(), map[string]string{
			"nome":  "João Silva",
			"email": "joao@email.com",
			"senha": "123",
		})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("retorna 409 quando email já está cadastrado", func(t *testing.T) {
		h := novoHandler()
		body := map[string]string{
			"nome":  "João Silva",
			"email": "joao@email.com",
			"senha": "12345678",
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
