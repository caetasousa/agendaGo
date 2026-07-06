package token_test

import (
	"testing"

	"agendago/internal/pkg/token"
)

func TestGerar(t *testing.T) {
	t.Run("gera tokens diferentes a cada chamada", func(t *testing.T) {
		t1, err := token.Gerar()
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		t2, err := token.Gerar()
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if t1 == t2 {
			t.Error("esperava tokens diferentes")
		}
		if t1 == "" {
			t.Error("token não deve ser vazio")
		}
	})
}

func TestHash(t *testing.T) {
	t.Run("gera o mesmo hash para o mesmo token", func(t *testing.T) {
		tok, _ := token.Gerar()
		if token.Hash(tok) != token.Hash(tok) {
			t.Error("esperava hash determinístico para o mesmo token")
		}
	})

	t.Run("gera hashes diferentes para tokens diferentes", func(t *testing.T) {
		t1, _ := token.Gerar()
		t2, _ := token.Gerar()
		if token.Hash(t1) == token.Hash(t2) {
			t.Error("esperava hashes diferentes para tokens diferentes")
		}
	})

	t.Run("hash em hexadecimal de 64 caracteres (SHA-256)", func(t *testing.T) {
		tok, _ := token.Gerar()
		h := token.Hash(tok)
		if len(h) != 64 {
			t.Errorf("esperava 64 caracteres, got: %d", len(h))
		}
	})
}
