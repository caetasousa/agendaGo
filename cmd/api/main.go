// Package main é o entrypoint do servidor HTTP.
//
//	@title			agendaGo API
//	@version		0.1.0
//	@description	API de agendamento entre clientes e prestadores de serviço.
//	@host			localhost:8080
//	@BasePath		/
package main

import (
	"encoding/json"
	"net/http"

	_ "agendago/docs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/health", health)
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	http.ListenAndServe(":8080", r)
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
