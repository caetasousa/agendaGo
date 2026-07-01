package config

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Porta define o endereço em que o servidor HTTP escuta.
const Porta = ":8080"

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
	r.Get("/swagger/*", httpSwagger.WrapHandler)
	return r
}
