//go:build !swagger

// Ausência da doc interativa — o estado padrão, e o que vai para produção.
//
// A doc é ferramenta de desenvolvimento: publica a superfície inteira da API
// (rotas, formatos, exemplos) para quem alcançar a porta. Antes isto era um
// `if !EhProducao()` em tempo de execução, o que resolvia a exposição mas
// deixava 10 MB de assets da UI e da spec dentro do binário de produção — 37%
// dele, para servir uma rota que nunca existia ali.
//
// Virou decisão de compilação por dois motivos: o código simplesmente não entra
// no binário, e não há variável de ambiente que possa ligá-lo por engano.
package config

import "github.com/go-chi/chi/v5"

// montarSwagger não faz nada nesta variante: a doc não foi compilada.
func montarSwagger(*chi.Mux) {}
