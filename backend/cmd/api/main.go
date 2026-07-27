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
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"agendago/config"
	_ "agendago/docs"
	"agendago/internal/adapter/email"
	"agendago/internal/adapter/http/handler"
	"agendago/internal/adapter/http/middleware"
	"agendago/internal/adapter/oauth"
	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	"agendago/internal/adapter/worker"
	"agendago/internal/pkg/logging"
	ucadmin "agendago/internal/usecase/admin"
	ucappointment "agendago/internal/usecase/appointment"
	ucauth "agendago/internal/usecase/auth"
	ucavailability "agendago/internal/usecase/availability"
	ucclient "agendago/internal/usecase/client"
	ucprovider "agendago/internal/usecase/provider"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

func main() {
	// logger estruturado (slog): JSON em produção, texto legível em dev.
	// Configurado antes de tudo, para todo log já sair no formato certo.
	logging.Configurar(config.EhProducao())

	// contexto de vida da aplicação: cancelado em SIGINT/SIGTERM, usado pelo
	// desligamento gracioso do servidor HTTP e do worker de lembretes
	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

	// banco de dados
	pool, err := config.NovoPool(context.Background())
	if err != nil {
		slog.Error("erro ao conectar no banco", slog.String("erro", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// repositórios
	providerRepo := repository.NovoProviderPostgres(pool)
	clientRepo := repository.NovoClientPostgres(pool)
	sessionRepo := repository.NovoSessionPostgres(pool)
	availabilityRepo := repository.NovoAvailabilityPostgres(pool)
	appointmentRepo := repository.NovoAppointmentPostgres(pool)
	adminRepo := repository.NovoAdminPostgres(pool)
	passwordResetRepo := repository.NovoPasswordResetPostgres(pool)
	cancelamentoRepo := repository.NovoCancellationPostgres(pool)
	signupRepo := repository.NovoSignupPostgres(pool)
	preCadastroRepo := repository.NovoPreCadastroPostgres(pool)
	socialIdentityRepo := repository.NovoSocialIdentityPostgres(pool)
	oauthStateRepo := repository.NovoOAuthStatePostgres(pool)

	// segurança
	hasher := security.NovoHasherArgon2id()

	// email: WaitGroup compartilhado entre os envios assíncronos e o worker
	// de lembretes, para o desligamento gracioso esperar o que estiver pendente
	var tarefasEmFundo sync.WaitGroup
	mailer := novoMailer()
	notificador := email.NovoNotificador(mailer, config.OrigemFrontend(), config.FusoHorario, email.ExecutorGoroutine(&tarefasEmFundo))

	// semente do admin (idempotente): cria/atualiza a partir das env vars
	if err := ucadmin.NovoSemearUseCase(adminRepo, hasher).Executar(config.AdminEmail(), config.AdminSenha()); err != nil {
		slog.Error("erro ao semear admin", slog.String("erro", err.Error()))
		os.Exit(1)
	}

	// aviso alto-visibilidade: rate limit desligado (RATE_LIMIT_*=0) some com
	// o teto de login, cadastro e brute-force de token — fácil de esquecer
	// ligado numa env de deploy copiada de dev
	desligados := make([]string, 0, 4)
	for nome, limite := range map[string]int{
		"RATE_LIMIT_LOGIN_POR_MINUTO":       config.RateLimitLoginPorMinuto(),
		"RATE_LIMIT_CONVIDADO_POR_MINUTO":   config.RateLimitConvidadoPorMinuto(),
		"RATE_LIMIT_LOGIN_POR_CONTA":        config.RateLimitLoginPorConta(),
		"RATE_LIMIT_AUTENTICADO_POR_MINUTO": config.RateLimitAutenticadoPorMinuto(),
		"RATE_LIMIT_PUBLICO_POR_MINUTO":     config.RateLimitPublicoPorMinuto(),
	} {
		if limite == 0 {
			desligados = append(desligados, nome)
		}
	}
	if len(desligados) > 0 {
		sort.Strings(desligados)
		slog.Warn("rate limiting parcialmente desligado: as rotas cobertas ficam sem teto",
			slog.String("variaveis_em_zero", strings.Join(desligados, ", ")))
	}

	// usecases
	solicitarCadastroProvider := ucprovider.NovoSolicitarCadastroUseCase(providerRepo, clientRepo, signupRepo, notificador, hasher)
	confirmarCadastroProvider := ucprovider.NovoConfirmarCadastroUseCase(providerRepo, clientRepo, signupRepo)
	atualizarPreferencias := ucprovider.NovoAtualizarPreferenciasUseCase(providerRepo)
	solicitarCadastroClient := ucclient.NovoSolicitarCadastroUseCase(clientRepo, providerRepo, signupRepo, notificador, hasher)
	confirmarCadastroClient := ucclient.NovoConfirmarCadastroUseCase(clientRepo, providerRepo, signupRepo)
	consultarPreCadastro := ucclient.NovoConsultarPreCadastroUseCase(preCadastroRepo)
	concluirPreCadastro := ucclient.NovoConcluirPreCadastroUseCase(clientRepo, providerRepo, preCadastroRepo, hasher)
	loginProvider := ucauth.NovoLoginProviderUseCase(providerRepo, sessionRepo, hasher)
	loginClient := ucauth.NovoLoginClientUseCase(clientRepo, sessionRepo, hasher)
	loginAdmin := ucauth.NovoLoginAdminUseCase(adminRepo, sessionRepo, hasher)
	loginSocial := novoLoginSocialUseCase(context.Background(), providerRepo, clientRepo, socialIdentityRepo, oauthStateRepo, sessionRepo, hasher)
	logout := ucauth.NovoLogoutUseCase(sessionRepo)
	validarSessao := ucauth.NovoValidarSessaoUseCase(sessionRepo)
	perfil := ucauth.NovoPerfilUseCase(providerRepo, clientRepo, adminRepo)
	solicitarRecuperacao := ucauth.NovoSolicitarRecuperacaoUseCase(providerRepo, clientRepo, passwordResetRepo, notificador)
	redefinirSenha := ucauth.NovoRedefinirSenhaUseCase(providerRepo, clientRepo, passwordResetRepo, sessionRepo, hasher)
	moderar := ucadmin.NovoModerarUseCase(providerRepo, clientRepo, sessionRepo)
	consultarAgenda := ucavailability.NovoConsultarAgendaUseCase(availabilityRepo, providerRepo)
	definirDia := ucavailability.NovoDefinirDiaUseCase(availabilityRepo)
	removerDia := ucavailability.NovoRemoverDiaUseCase(availabilityRepo)
	consultarDisponibilidade := ucavailability.NovoConsultarDisponibilidadeUseCase(availabilityRepo, providerRepo)
	listarPrestadores := ucprovider.NovoListarUseCase(providerRepo)
	buscarPrestador := ucprovider.NovoBuscarResumoUseCase(providerRepo)
	consultarSlots := ucappointment.NovoConsultarSlotsUseCase(consultarDisponibilidade, appointmentRepo, providerRepo, config.FusoHorario)
	solicitarAgendamento := ucappointment.NovoSolicitarUseCase(consultarSlots, appointmentRepo, clientRepo, providerRepo, notificador, config.TTLSolicitacao)
	solicitarConvidado := ucappointment.NovoSolicitarConvidadoUseCase(solicitarAgendamento, clientRepo, providerRepo, cancelamentoRepo, preCadastroRepo, notificador)
	marcarPeloPrestador := ucappointment.NovoMarcarPeloPrestadorUseCase(solicitarAgendamento, clientRepo, providerRepo)
	transicionarAgendamento := ucappointment.NovoTransicionarUseCase(appointmentRepo, providerRepo, clientRepo, cancelamentoRepo, preCadastroRepo, notificador, config.AntecedenciaMinimaCancelamento, config.FusoHorario)
	cancelarPorToken := ucappointment.NovoCancelarPorTokenUseCase(appointmentRepo, cancelamentoRepo, providerRepo, clientRepo, notificador, config.AntecedenciaMinimaCancelamento, config.FusoHorario)
	listarAgendamentos := ucappointment.NovoListarUseCase(appointmentRepo, providerRepo, clientRepo)
	detalharUsuario := ucadmin.NovoDetalharUseCase(providerRepo, clientRepo, listarAgendamentos)
	lembrarAgendamento := ucappointment.NovoLembrarUseCase(appointmentRepo, providerRepo, clientRepo, notificador, config.FusoHorario, config.AntecedenciaLembrete)

	// handlers
	identidadeDoContexto := func(r *http.Request) (ucauth.Identidade, bool) {
		return middleware.IdentidadeDoContexto(r.Context())
	}
	providerHandler := handler.NovoProviderHandler(solicitarCadastroProvider, confirmarCadastroProvider, atualizarPreferencias, listarPrestadores, buscarPrestador, identidadeDoContexto)
	clientHandler := handler.NovoClientHandler(solicitarCadastroClient, confirmarCadastroClient, consultarPreCadastro, concluirPreCadastro)
	// teto de tentativas por conta, compartilhado entre login e recuperação de
	// senha: o mesmo contador, com chaves de prefixo diferente
	limitadorPorConta := handler.NovoLimitadorPorConta(config.RateLimitLoginPorConta(), config.JanelaLimitePorConta)
	authHandler := handler.NovoAuthHandler(loginProvider, loginClient, loginAdmin, logout, perfil, config.CookieSeguro(), limitadorPorConta, identidadeDoContexto)
	oauthHandler := handler.NovoOAuthHandler(loginSocial, config.CookieSeguro(), config.OrigemFrontend())
	passwordResetHandler := handler.NovoPasswordResetHandler(solicitarRecuperacao, redefinirSenha, limitadorPorConta)
	availabilityHandler := handler.NovoAvailabilityHandler(consultarAgenda, definirDia, removerDia, identidadeDoContexto)
	appointmentHandler := handler.NovoAppointmentHandler(consultarSlots, solicitarAgendamento, solicitarConvidado, marcarPeloPrestador, transicionarAgendamento, cancelarPorToken, listarAgendamentos, identidadeDoContexto)
	adminHandler := handler.NovoAdminHandler(moderar, detalharUsuario)

	// middlewares
	authMw := middleware.NovoAuth(validarSessao, config.CookieSeguro())

	// Os tetos de requisição do projeto, todos com 0 = desligado. Por IP para
	// quem ainda não se identificou; por sessão depois do login, quando o IP
	// deixa de dizer alguma coisa (e o teto por conta, o terceiro, vive dentro
	// dos handlers de login e recuperação de senha).
	limitarPorIP := func(r chi.Router, limite int) {
		if limite > 0 {
			r.Use(httprate.LimitBy(limite, time.Minute, middleware.ChavePorIP,
				httprate.WithLimitHandler(middleware.RespostaLimiteExcedido)))
		}
	}
	limitarPorSessao := func(r chi.Router, limite int) {
		if limite > 0 {
			r.Use(httprate.LimitBy(limite, time.Minute, middleware.ChavePorSessao,
				httprate.WithLimitHandler(middleware.RespostaLimiteExcedido)))
		}
	}

	// roteador
	r := config.NovoRouter()
	r.Get("/health", health)
	// leituras públicas: respondem a qualquer um, sem identificação nenhuma.
	// O teto evita que raspar a vitrine inteira em laço custe ao banco.
	r.Group(func(r chi.Router) {
		limitarPorIP(r, config.RateLimitPublicoPorMinuto())
		r.Get("/providers", providerHandler.Listar)
		r.Get("/providers/{id}", providerHandler.BuscarResumo)
		r.Get("/providers/{id}/slots", appointmentHandler.ConsultarSlots)
	})
	// rotas públicas de convidado (agendar e cancelar por token) têm teto por
	// IP: sem ele, uma rajada enche a agenda de um prestador com reservas
	// falsas ou tenta adivinhar tokens de cancelamento por força bruta
	r.Group(func(r chi.Router) {
		limitarPorIP(r, config.RateLimitConvidadoPorMinuto())
		// o detalhe do agendamento vem por token no path: nada disso pode
		// ficar no cache do navegador de um computador compartilhado
		r.Use(middleware.SemCache)
		r.Post("/agendamentos/convidado", appointmentHandler.SolicitarConvidado)
		r.Get("/agendamentos/cancelar/{token}", appointmentHandler.DetalharCancelamento)
		r.Post("/agendamentos/cancelar/{token}", appointmentHandler.CancelarPorToken)
	})
	// cadastro (prestador e cliente) roda Argon2id (custo de CPU/memória por
	// request) e o de cliente ainda dispara email: teto por IP mitiga DoS de
	// hashing, spam de emails e força bruta de token de confirmação
	r.Group(func(r chi.Router) {
		limitarPorIP(r, config.RateLimitLoginPorMinuto())
		// o pré-cadastro devolve nome/email/telefone a partir de um token no
		// path — resposta pessoal, fora do cache do navegador
		r.Use(middleware.SemCache)
		r.Post("/providers", providerHandler.Cadastrar)
		r.Post("/providers/confirmar-cadastro", providerHandler.ConfirmarCadastro)
		r.Post("/clients", clientHandler.Cadastrar)
		r.Post("/clients/confirmar-cadastro", clientHandler.ConfirmarCadastro)
		r.Get("/clients/pre-cadastro/{token}", clientHandler.ConsultarPreCadastro)
		r.Post("/clients/pre-cadastro/{token}", clientHandler.ConcluirPreCadastro)
	})
	// logins têm teto por IP: mitiga brute-force e rajadas de Argon2id (CPU).
	// O teto por CONTA, que pega quem troca de IP a cada tentativa, está dentro
	// do handler — só lá o email já foi lido do corpo.
	r.Group(func(r chi.Router) {
		limitarPorIP(r, config.RateLimitLoginPorMinuto())
		r.Post("/auth/provider/login", authHandler.LoginProvider)
		r.Post("/auth/client/login", authHandler.LoginClient)
		r.Post("/auth/admin/login", authHandler.LoginAdmin)
		// login social (Google): rotas só registradas com credenciais
		// configuradas, para não expor um fluxo que vai falhar sempre
		if config.OAuthGoogleAtivo() {
			// entrada da tela de login: sem tipo, porque a conta já existe e
			// o sistema descobre sozinho se é cliente ou prestador
			r.Get("/auth/google/start", oauthHandler.GoogleStartLogin)
			// entradas do cadastro, onde o tipo é escolhido antes
			r.Get("/auth/client/google/start", oauthHandler.GoogleStartClient)
			r.Get("/auth/provider/google/start", oauthHandler.GoogleStartProvider)
			r.Get("/auth/google/callback", oauthHandler.GoogleCallback)
		}
	})
	r.Post("/auth/logout", authHandler.Logout)
	// recuperação de senha tem teto por IP, como os logins: mitiga farming de
	// tokens e envio abusivo de emails
	r.Group(func(r chi.Router) {
		limitarPorIP(r, config.RateLimitLoginPorMinuto())
		r.Post("/auth/recuperar-senha", passwordResetHandler.Solicitar)
		r.Post("/auth/redefinir-senha", passwordResetHandler.Redefinir)
	})
	r.Group(func(r chi.Router) {
		r.Use(middleware.SemCache)
		r.Use(authMw.Autenticar)
		r.Get("/auth/me", authHandler.Me)
	})
	r.Group(func(r chi.Router) {
		r.Use(middleware.SemCache)
		r.Use(authMw.Autenticar)
		limitarPorSessao(r, config.RateLimitAutenticadoPorMinuto())
		r.Use(middleware.ExigirProvider)
		r.Put("/providers/me/preferencias", providerHandler.AtualizarPreferencias)
		r.Get("/providers/me/agenda", availabilityHandler.ConsultarAgenda)
		r.Put("/providers/me/dias/{data}", availabilityHandler.DefinirDia)
		r.Delete("/providers/me/dias/{data}", availabilityHandler.RemoverDia)
		r.Get("/providers/me/agendamentos", appointmentHandler.ListarDoPrestador)
		// marcação feita pelo próprio prestador (cliente ligou): slots da
		// própria agenda (mesmo fechada ao público) e o registro da reserva
		r.Get("/providers/me/slots", appointmentHandler.ConsultarSlotsDoPrestador)
		r.Post("/providers/me/agendamentos", appointmentHandler.MarcarPeloPrestador)
	})
	r.Group(func(r chi.Router) {
		r.Use(middleware.SemCache)
		r.Use(authMw.Autenticar)
		limitarPorSessao(r, config.RateLimitAutenticadoPorMinuto())
		r.Use(middleware.ExigirClient)
		r.Post("/agendamentos", appointmentHandler.Solicitar)
		r.Get("/clients/me/agendamentos", appointmentHandler.ListarDoCliente)
	})
	r.Group(func(r chi.Router) {
		r.Use(middleware.SemCache)
		r.Use(authMw.Autenticar)
		limitarPorSessao(r, config.RateLimitAutenticadoPorMinuto())
		r.Post("/agendamentos/{id}/confirmar", appointmentHandler.Confirmar)
		r.Post("/agendamentos/{id}/recusar", appointmentHandler.Recusar)
		r.Post("/agendamentos/{id}/cancelar", appointmentHandler.Cancelar)
		r.Post("/agendamentos/{id}/realizado", appointmentHandler.MarcarRealizado)
		r.Post("/agendamentos/{id}/nao-compareceu", appointmentHandler.MarcarNaoCompareceu)
	})
	r.Group(func(r chi.Router) {
		r.Use(middleware.SemCache)
		r.Use(authMw.Autenticar)
		limitarPorSessao(r, config.RateLimitAutenticadoPorMinuto())
		r.Use(middleware.ExigirAdmin)
		r.Get("/admin/prestadores", adminHandler.ListarPrestadores)
		r.Get("/admin/prestadores/{id}", adminHandler.DetalharPrestador)
		r.Get("/admin/clientes", adminHandler.ListarClientes)
		r.Get("/admin/clientes/{id}", adminHandler.DetalharCliente)
		r.Post("/admin/prestadores/{id}/banir", adminHandler.BanirPrestador)
		r.Post("/admin/prestadores/{id}/reativar", adminHandler.ReativarPrestador)
		r.Post("/admin/clientes/{id}/banir", adminHandler.BanirCliente)
		r.Post("/admin/clientes/{id}/reativar", adminHandler.ReativarCliente)
	})

	// worker de lembretes: roda em segundo plano até o contexto ser cancelado
	reminderWorker := worker.NovoReminderWorker(lembrarAgendamento, config.IntervaloVerificacaoLembrete)
	tarefasEmFundo.Add(1)
	go func() {
		defer tarefasEmFundo.Done()
		reminderWorker.Executar(ctx)
	}()

	// worker de limpeza: apaga tokens expirados que ninguém mais vai consumir
	cleanupWorker := worker.NovoCleanupWorker(config.IntervaloLimpezaTokens, signupRepo, passwordResetRepo, preCadastroRepo, cancelamentoRepo)
	tarefasEmFundo.Add(1)
	go func() {
		defer tarefasEmFundo.Done()
		cleanupWorker.Executar(ctx)
	}()

	// servidor com desligamento gracioso: SIGINT/SIGTERM param de aceitar
	// conexões novas e as requisições em andamento têm um prazo para concluir
	srv := config.NovoServidor(r)
	go func() {
		slog.Info("servidor iniciado", slog.String("porta", config.Porta()))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("erro ao iniciar servidor", slog.String("erro", err.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	slog.Info("encerrando: aguardando requisições em andamento")
	ctxDesligamento, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	if err := srv.Shutdown(ctxDesligamento); err != nil {
		slog.Error("desligamento forçado", slog.String("erro", err.Error()))
	}

	// espera o worker parar e os emails assíncronos pendentes terminarem
	tarefasEmFundo.Wait()
	slog.Info("servidor encerrado")
}

// novoMailer cria o transporte de email: SMTP real quando configurado
// (config.EmailAtivo), ou um mailer nulo que só loga — assim o boot e os
// ambientes sem SMTP configurado não quebram.
func novoMailer() interface{ Enviar(email.Mensagem) error } {
	if !config.EmailAtivo() {
		return email.MailerNulo{}
	}
	m, err := email.NovaMailerSMTP(
		config.SMTPHost(), config.SMTPPort(), config.SMTPUser(), config.SMTPPassword(),
		config.SMTPStartTLS(), config.EmailRemetente(), config.EmailRemetenteNome(), config.EmailReplyTo(),
	)
	if err != nil {
		slog.Error("erro ao configurar SMTP", slog.String("erro", err.Error()))
		os.Exit(1)
	}
	return m
}

// novoLoginSocialUseCase cria o usecase de login social com o adapter Google
// já configurado, ou nil quando as credenciais OAuth não estão definidas —
// nesse caso o roteador não registra as rotas correspondentes (ver
// config.OAuthGoogleAtivo). O discovery do endpoint OIDC do Google acontece
// aqui, no boot, para falhar cedo em vez de a cada tentativa de login.
func novoLoginSocialUseCase(
	ctx context.Context,
	providerRepo *repository.ProviderPostgres,
	clientRepo *repository.ClientPostgres,
	socialIdentityRepo *repository.SocialIdentityPostgres,
	oauthStateRepo *repository.OAuthStatePostgres,
	sessionRepo *repository.SessionPostgres,
	hasher *security.HasherArgon2id,
) *ucauth.LoginSocialUseCase {
	if !config.OAuthGoogleAtivo() {
		return nil
	}

	google, err := oauth.NovoGoogle(ctx, config.GoogleClientID(), config.GoogleClientSecret(), config.GoogleRedirectURL())
	if err != nil {
		slog.Error("erro ao configurar login social com Google", slog.String("erro", err.Error()))
		os.Exit(1)
	}

	return ucauth.NovoLoginSocialUseCase(google, clientRepo, providerRepo, clientRepo, providerRepo, socialIdentityRepo, oauthStateRepo, sessionRepo, hasher)
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
