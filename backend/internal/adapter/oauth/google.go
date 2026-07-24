// Package oauth implementa os adapters de login social (OIDC) por trás do
// port ProvedorOIDC declarado em usecase/auth.
package oauth

import (
	"context"
	"fmt"

	"agendago/internal/domain/socialidentity"

	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"

	"github.com/coreos/go-oidc/v3/oidc"
)

// issuerGoogle é o issuer OIDC do Google, usado para descobrir o endpoint e o
// JWKS de verificação do id_token.
const issuerGoogle = "https://accounts.google.com"

// escoposLoginGoogle pede só identidade — nenhum escopo de calendário. O
// consentimento de calendário é um fluxo OAuth separado (fora desta fase).
var escoposLoginGoogle = []string{oidc.ScopeOpenID, "email", "profile"}

// Google é o adapter de login social via Sign in with Google.
type Google struct {
	config   oauth2.Config
	provider *oidc.Provider
}

// NovoGoogle cria o adapter Google, descobrindo o endpoint OIDC (autorização,
// token e JWKS) a partir do issuer. Falha se o discovery document não puder
// ser obtido — chamar no boot, não sob demanda.
func NovoGoogle(ctx context.Context, clientID, clientSecret, redirectURL string) (*Google, error) {
	provider, err := oidc.NewProvider(ctx, issuerGoogle)
	if err != nil {
		return nil, fmt.Errorf("descobrir provedor OIDC do Google: %w", err)
	}

	return &Google{
		provider: provider,
		config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     googleoauth.Endpoint,
			Scopes:       escoposLoginGoogle,
		},
	}, nil
}

// URLAutorizacao monta a URL de consentimento do Google. state e nonce
// protegem contra CSRF e replay do id_token, respectivamente.
func (g *Google) URLAutorizacao(state, nonce string) string {
	return g.config.AuthCodeURL(state, oidc.Nonce(nonce))
}

// TrocarCodigo troca o código de autorização pelo token, extrai o id_token e
// o verifica contra o JWKS do Google (assinatura, issuer, audience e nonce).
// Retorna erro se o código for inválido ou o id_token não verificar.
func (g *Google) TrocarCodigo(ctx context.Context, code, nonceEsperado string) (*socialidentity.IdentidadeOIDC, error) {
	tok, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("trocar código por token: %w", err)
	}

	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, fmt.Errorf("resposta do Google sem id_token")
	}

	verifier := g.provider.Verifier(&oidc.Config{ClientID: g.config.ClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("verificar id_token: %w", err)
	}
	if idToken.Nonce != nonceEsperado {
		return nil, fmt.Errorf("nonce do id_token não confere")
	}

	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("ler claims do id_token: %w", err)
	}

	return &socialidentity.IdentidadeOIDC{
		Sub:             claims.Sub,
		Email:           claims.Email,
		EmailVerificado: claims.EmailVerified,
		Nome:            claims.Name,
	}, nil
}
