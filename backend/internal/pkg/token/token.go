// Package token gera e deriva tokens opacos usados nas sessões de autenticação.
package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// tamanhoBytes é o tamanho do token aleatório antes da codificação (256 bits).
const tamanhoBytes = 32

// Gerar cria um token aleatório de 256 bits codificado em base64url.
func Gerar() (string, error) {
	b := make([]byte, tamanhoBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Hash devolve o SHA-256 do token em hexadecimal — é essa representação que
// fica persistida no banco, nunca o token puro.
func Hash(t string) string {
	soma := sha256.Sum256([]byte(t))
	return hex.EncodeToString(soma[:])
}
