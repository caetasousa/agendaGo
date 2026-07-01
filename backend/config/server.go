package config

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Porta define o endereço em que o servidor HTTP escuta.
const Porta = ":8080"

// OrigemFrontend é a origem permitida do frontend em desenvolvimento.
const OrigemFrontend = "http://localhost:5173"

// Servidor encapsula o http.Server com as configurações do projeto.
type Servidor struct {
	http.Server
}

// NovoServidor cria um Servidor HTTP com timeouts configurados.
func NovoServidor(r *chi.Mux) *Servidor {
	return &Servidor{
		Server: http.Server{
			Addr:         Porta,
			Handler:      r,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// NovoRouter cria um roteador chi com middlewares de log, recuperação de panics e swagger.
func NovoRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{OrigemFrontend},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}))
	r.Get("/swagger/*", httpSwagger.WrapHandler)
	return r
}
