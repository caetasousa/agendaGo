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
	"agendago/internal/adapter/repository"
	ucprovider "agendago/internal/usecase/provider"
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

	// usecases
	cadastrarProvider := ucprovider.NovoCadastrarUseCase(providerRepo)

	// handlers
	providerHandler := handler.NovoProviderHandler(cadastrarProvider)

	// roteador
	r := config.NovoRouter()
	r.Get("/health", health)
	r.Post("/providers", providerHandler.Cadastrar)

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
