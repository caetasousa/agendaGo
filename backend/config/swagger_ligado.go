//go:build swagger

// Monta a doc interativa. Compilado só com `-tags=swagger`, que é como as
// imagens de desenvolvimento constroem o binário — ver Dockerfile e .air.toml.
package config

import (
	// A spec gerada por swag registra-se no pacote pelo efeito do import. É
	// aqui, e não no main.go, justamente para que o binário de produção não a
	// carregue: são 114 KB de JSON como string, mais os assets da UI que o
	// swaggo/files embute.
	_ "agendago/docs"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

// montarSwagger registra a rota da doc interativa no roteador.
func montarSwagger(r *chi.Mux) {
	r.Get("/swagger/*", httpSwagger.WrapHandler)
}
