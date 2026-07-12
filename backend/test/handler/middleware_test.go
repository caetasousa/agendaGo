package handler_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agendago/internal/adapter/http/middleware"
)

func TestLimitarCorpo(t *testing.T) {
	// handler que tenta consumir o corpo inteiro, como os decoders de JSON fazem
	h := middleware.LimitarCorpo(16)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			http.Error(w, "corpo grande demais", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("corpo dentro do limite passa", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("pequeno")))
		if rr.Code != http.StatusOK {
			t.Errorf("esperava 200, got: %d", rr.Code)
		}
	})

	t.Run("corpo acima do limite é barrado", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 64))))
		if rr.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("esperava 413, got: %d", rr.Code)
		}
	})
}
