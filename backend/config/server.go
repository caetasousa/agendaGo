package config

import (
	"net/http"
	"os"
	"strconv"
	"time"

	appmiddleware "agendago/internal/adapter/http/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Porta devolve o endereço em que o servidor HTTP escuta
// (env PORT, padrão 8080).
func Porta() string {
	if p := os.Getenv("PORT"); p != "" {
		return ":" + p
	}
	return ":8080"
}

// OrigemFrontend devolve a origem permitida no CORS (env FRONTEND_ORIGIN).
// O padrão é o frontend de desenvolvimento; em produção aponte para o domínio real.
func OrigemFrontend() string {
	if o := os.Getenv("FRONTEND_ORIGIN"); o != "" {
		return o
	}
	return "http://localhost:5173"
}

// CookieSeguro informa se o cookie de sessão deve ter o atributo Secure.
// Em desenvolvimento (http://localhost) o navegador nem sempre entrega
// cookies Secure de forma confiável, por isso o atributo só é ativado em produção.
func CookieSeguro() bool {
	return os.Getenv("APP_ENV") == "production"
}

// AdminEmail e AdminSenha são as credenciais do administrador semeado no boot.
// Vazias significam "sem admin" — nenhuma conta de moderação é criada.
func AdminEmail() string { return os.Getenv("ADMIN_EMAIL") }
func AdminSenha() string { return os.Getenv("ADMIN_SENHA") }

// RateLimitLoginPorMinuto é o teto de tentativas de login por IP por minuto
// (env RATE_LIMIT_LOGIN_POR_MINUTO; 0 desliga). Mitiga brute-force e o custo
// de CPU do Argon2id em rajadas.
func RateLimitLoginPorMinuto() int {
	return intDoAmbiente("RATE_LIMIT_LOGIN_POR_MINUTO", 10)
}

// RateLimitConvidadoPorMinuto é o teto de agendamentos de convidado por IP por
// minuto (env RATE_LIMIT_CONVIDADO_POR_MINUTO; 0 desliga). A rota é pública —
// sem teto, uma rajada enche a agenda de um prestador com reservas falsas.
func RateLimitConvidadoPorMinuto() int {
	return intDoAmbiente("RATE_LIMIT_CONVIDADO_POR_MINUTO", 10)
}

// intDoAmbiente lê um inteiro não negativo da env var, caindo no padrão quando
// ausente ou inválida.
func intDoAmbiente(nome string, padrao int) int {
	v := os.Getenv(nome)
	if v == "" {
		return padrao
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return padrao
	}
	return n
}

// Servidor encapsula o http.Server com as configurações do projeto.
type Servidor struct {
	http.Server
}

// NovoServidor cria um Servidor HTTP com timeouts configurados.
func NovoServidor(r *chi.Mux) *Servidor {
	return &Servidor{
		Server: http.Server{
			Addr:           Porta(),
			Handler:        r,
			ReadTimeout:    15 * time.Second,
			WriteTimeout:   15 * time.Second,
			IdleTimeout:    60 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
	}
}

// maxBytesCorpo limita o corpo de qualquer requisição — a API só troca JSONs pequenos.
const maxBytesCorpo = 1 << 20 // 1 MiB

// NovoRouter cria um roteador chi com middlewares de log, recuperação de
// panics, limite de corpo e swagger.
func NovoRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(appmiddleware.LimitarCorpo(maxBytesCorpo))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{OrigemFrontend()},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}))
	r.Get("/swagger/*", httpSwagger.WrapHandler)
	return r
}
