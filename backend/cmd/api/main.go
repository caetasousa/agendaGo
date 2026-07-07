// Package main é o entrypoint do servidor HTTP.
//
//	@title			agendaGo API
//	@version		0.1.0
//	@description	API de agendamento entre clientes e prestadores de serviço.
//	@host			localhost:8080
//	@BasePath		/
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	_ "agendago/docs"
	"agendago/config"
	"agendago/internal/adapter/http/handler"
	"agendago/internal/adapter/http/middleware"
	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	ucauth "agendago/internal/usecase/auth"
	ucclient "agendago/internal/usecase/client"
	ucprovider "agendago/internal/usecase/provider"

	"github.com/go-chi/chi/v5"
)

func main() {
	// banco de dados
	pool, err := config.NovoPool(context.Background())
	if err != nil {
		log.Fatalf("erro ao conectar no banco: %v", err)
	}
	defer pool.Close()

	// repositórios
	providerRepo := repository.NovoProviderPostgres(pool)
	clientRepo := repository.NovoClientPostgres(pool)
	sessionRepo := repository.NovoSessionPostgres(pool)

	// segurança
	hasher := security.NovoHasherArgon2id()

	// usecases
	cadastrarProvider := ucprovider.NovoCadastrarUseCase(providerRepo, hasher)
	atualizarPreferencias := ucprovider.NovoAtualizarPreferenciasUseCase(providerRepo)
	cadastrarClient := ucclient.NovoCadastrarUseCase(clientRepo, hasher)
	loginProvider := ucauth.NovoLoginProviderUseCase(providerRepo, sessionRepo, hasher)
	loginClient := ucauth.NovoLoginClientUseCase(clientRepo, sessionRepo, hasher)
	logout := ucauth.NovoLogoutUseCase(sessionRepo)
	validarSessao := ucauth.NovoValidarSessaoUseCase(sessionRepo)
	perfil := ucauth.NovoPerfilUseCase(providerRepo, clientRepo)

	// handlers
	identidadeDoContexto := func(r *http.Request) (ucauth.Identidade, bool) {
		return middleware.IdentidadeDoContexto(r.Context())
	}
	providerHandler := handler.NovoProviderHandler(cadastrarProvider, atualizarPreferencias, identidadeDoContexto)
	clientHandler := handler.NovoClientHandler(cadastrarClient)
	authHandler := handler.NovoAuthHandler(loginProvider, loginClient, logout, perfil, config.CookieSeguro(), identidadeDoContexto)

	// middlewares
	authMw := middleware.NovoAuth(validarSessao)

	// roteador
	r := config.NovoRouter()
	r.Get("/health", health)
	r.Post("/providers", providerHandler.Cadastrar)
	r.Post("/clients", clientHandler.Cadastrar)
	r.Post("/auth/provider/login", authHandler.LoginProvider)
	r.Post("/auth/client/login", authHandler.LoginClient)
	r.Post("/auth/logout", authHandler.Logout)
	r.Group(func(r chi.Router) {
		r.Use(authMw.Autenticar)
		r.Get("/auth/me", authHandler.Me)
	})
	r.Group(func(r chi.Router) {
		r.Use(authMw.Autenticar)
		r.Use(middleware.ExigirProvider)
		r.Put("/providers/me/preferencias", providerHandler.AtualizarPreferencias)
	})

	// servidor
	srv := config.NovoServidor(r)
	log.Printf("servidor iniciado na porta %s", config.Porta)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("erro ao iniciar servidor: %v", err)
	}
}

// health godoc
//
//	@Summary		Health check
//	@Description	Retorna o status do servidor
//	@Tags			infra
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Router			/health [get]
func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
