package security_test

import (
	"testing"

	"agendago/internal/adapter/security"
)

func TestHasherArgon2id(t *testing.T) {
	t.Run("gera hash e verifica com sucesso a senha correta", func(t *testing.T) {
		h := security.NovoHasherArgon2id()
		hash, err := h.Gerar("senha-secreta")
		if err != nil {
			t.Fatalf("esperava sucesso ao gerar hash, got: %v", err)
		}

		ok, err := h.Verificar("senha-secreta", hash)
		if err != nil {
			t.Fatalf("esperava sucesso ao verificar, got: %v", err)
		}
		if !ok {
			t.Error("esperava senha correta ser aceita")
		}
	})

	t.Run("rejeita senha incorreta", func(t *testing.T) {
		h := security.NovoHasherArgon2id()
		hash, _ := h.Gerar("senha-secreta")

		ok, err := h.Verificar("senha-errada", hash)
		if err != nil {
			t.Fatalf("não esperava erro, got: %v", err)
		}
		if ok {
			t.Error("esperava senha incorreta ser rejeitada")
		}
	})

	t.Run("gera hashes diferentes para a mesma senha (salt aleatório)", func(t *testing.T) {
		h := security.NovoHasherArgon2id()
		hash1, _ := h.Gerar("senha-secreta")
		hash2, _ := h.Gerar("senha-secreta")

		if hash1 == hash2 {
			t.Error("esperava hashes diferentes por causa do salt aleatório")
		}
	})
}
