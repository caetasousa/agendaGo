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
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"agendago/config"
	_ "agendago/docs"
	"agendago/internal/adapter/email"
	"agendago/internal/adapter/http/handler"
	"agendago/internal/adapter/http/middleware"
	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	"agendago/internal/adapter/worker"
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
	// contexto de vida da aplicação: cancelado em SIGINT/SIGTERM, usado pelo
	// desligamento gracioso do servidor HTTP e do worker de lembretes
	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

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
	availabilityRepo := repository.NovoAvailabilityPostgres(pool)
	appointmentRepo := repository.NovoAppointmentPostgres(pool)
	adminRepo := repository.NovoAdminPostgres(pool)
	passwordResetRepo := repository.NovoPasswordResetPostgres(pool)

	// segurança
	hasher := security.NovoHasherArgon2id()

	// email: WaitGroup compartilhado entre os envios assíncronos e o worker
	// de lembretes, para o desligamento gracioso esperar o que estiver pendente
	var tarefasEmFundo sync.WaitGroup
	mailer := novoMailer()
	notificador := email.NovoNotificador(mailer, config.OrigemFrontend(), config.FusoHorario, email.ExecutorGoroutine(&tarefasEmFundo))

	// semente do admin (idempotente): cria/atualiza a partir das env vars
	if err := ucadmin.NovoSemearUseCase(adminRepo, hasher).Executar(config.AdminEmail(), config.AdminSenha()); err != nil {
		log.Fatalf("erro ao semear admin: %v", err)
	}

	// usecases
	cadastrarProvider := ucprovider.NovoCadastrarUseCase(providerRepo, hasher)
	atualizarPreferencias := ucprovider.NovoAtualizarPreferenciasUseCase(providerRepo)
	cadastrarClient := ucclient.NovoCadastrarUseCase(clientRepo, hasher)
	loginProvider := ucauth.NovoLoginProviderUseCase(providerRepo, sessionRepo, hasher)
	loginClient := ucauth.NovoLoginClientUseCase(clientRepo, sessionRepo, hasher)
	loginAdmin := ucauth.NovoLoginAdminUseCase(adminRepo, sessionRepo, hasher)
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
	solicitarConvidado := ucappointment.NovoSolicitarConvidadoUseCase(solicitarAgendamento, clientRepo)
	transicionarAgendamento := ucappointment.NovoTransicionarUseCase(appointmentRepo, providerRepo, clientRepo, notificador, config.AntecedenciaMinimaCancelamento, config.FusoHorario)
	listarAgendamentos := ucappointment.NovoListarUseCase(appointmentRepo, providerRepo, clientRepo)
	detalharUsuario := ucadmin.NovoDetalharUseCase(providerRepo, clientRepo, listarAgendamentos)
	lembrarAgendamento := ucappointment.NovoLembrarUseCase(appointmentRepo, providerRepo, clientRepo, notificador, config.FusoHorario, config.AntecedenciaLembrete)

	// handlers
	identidadeDoContexto := func(r *http.Request) (ucauth.Identidade, bool) {
		return middleware.IdentidadeDoContexto(r.Context())
	}
	providerHandler := handler.NovoProviderHandler(cadastrarProvider, atualizarPreferencias, listarPrestadores, buscarPrestador, identidadeDoContexto)
	clientHandler := handler.NovoClientHandler(cadastrarClient)
	authHandler := handler.NovoAuthHandler(loginProvider, loginClient, loginAdmin, logout, perfil, config.CookieSeguro(), identidadeDoContexto)
	passwordResetHandler := handler.NovoPasswordResetHandler(solicitarRecuperacao, redefinirSenha)
	availabilityHandler := handler.NovoAvailabilityHandler(consultarAgenda, definirDia, removerDia, identidadeDoContexto)
	appointmentHandler := handler.NovoAppointmentHandler(consultarSlots, solicitarAgendamento, solicitarConvidado, transicionarAgendamento, listarAgendamentos, identidadeDoContexto)
	adminHandler := handler.NovoAdminHandler(moderar, detalharUsuario)

	// middlewares
	authMw := middleware.NovoAuth(validarSessao)

	// roteador
	r := config.NovoRouter()
	r.Get("/health", health)
	r.Get("/providers", providerHandler.Listar)
	r.Get("/providers/{id}", providerHandler.BuscarResumo)
	r.Get("/providers/{id}/slots", appointmentHandler.ConsultarSlots)
	// rota pública de convidado tem teto por IP: sem ele, uma rajada enche a
	// agenda de um prestador com reservas falsas
	r.Group(func(r chi.Router) {
		if limite := config.RateLimitConvidadoPorMinuto(); limite > 0 {
			r.Use(httprate.LimitByIP(limite, time.Minute))
		}
		r.Post("/agendamentos/convidado", appointmentHandler.SolicitarConvidado)
	})
	r.Post("/providers", providerHandler.Cadastrar)
	r.Post("/clients", clientHandler.Cadastrar)
	// logins têm teto por IP: mitiga brute-force e rajadas de Argon2id (CPU)
	r.Group(func(r chi.Router) {
		if limite := config.RateLimitLoginPorMinuto(); limite > 0 {
			r.Use(httprate.LimitByIP(limite, time.Minute))
		}
		r.Post("/auth/provider/login", authHandler.LoginProvider)
		r.Post("/auth/client/login", authHandler.LoginClient)
		r.Post("/auth/admin/login", authHandler.LoginAdmin)
	})
	r.Post("/auth/logout", authHandler.Logout)
	// recuperação de senha tem teto por IP, como os logins: mitiga farming de
	// tokens e envio abusivo de emails
	r.Group(func(r chi.Router) {
		if limite := config.RateLimitLoginPorMinuto(); limite > 0 {
			r.Use(httprate.LimitByIP(limite, time.Minute))
		}
		r.Post("/auth/recuperar-senha", passwordResetHandler.Solicitar)
		r.Post("/auth/redefinir-senha", passwordResetHandler.Redefinir)
	})
	r.Group(func(r chi.Router) {
		r.Use(authMw.Autenticar)
		r.Get("/auth/me", authHandler.Me)
	})
	r.Group(func(r chi.Router) {
		r.Use(authMw.Autenticar)
		r.Use(middleware.ExigirProvider)
		r.Put("/providers/me/preferencias", providerHandler.AtualizarPreferencias)
		r.Get("/providers/me/agenda", availabilityHandler.ConsultarAgenda)
		r.Put("/providers/me/dias/{data}", availabilityHandler.DefinirDia)
		r.Delete("/providers/me/dias/{data}", availabilityHandler.RemoverDia)
		r.Get("/providers/me/agendamentos", appointmentHandler.ListarDoPrestador)
	})
	r.Group(func(r chi.Router) {
		r.Use(authMw.Autenticar)
		r.Use(middleware.ExigirClient)
		r.Post("/agendamentos", appointmentHandler.Solicitar)
		r.Get("/clients/me/agendamentos", appointmentHandler.ListarDoCliente)
	})
	r.Group(func(r chi.Router) {
		r.Use(authMw.Autenticar)
		r.Post("/agendamentos/{id}/confirmar", appointmentHandler.Confirmar)
		r.Post("/agendamentos/{id}/recusar", appointmentHandler.Recusar)
		r.Post("/agendamentos/{id}/cancelar", appointmentHandler.Cancelar)
		r.Post("/agendamentos/{id}/realizado", appointmentHandler.MarcarRealizado)
		r.Post("/agendamentos/{id}/nao-compareceu", appointmentHandler.MarcarNaoCompareceu)
	})
	r.Group(func(r chi.Router) {
		r.Use(authMw.Autenticar)
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

	// servidor com desligamento gracioso: SIGINT/SIGTERM param de aceitar
	// conexões novas e as requisições em andamento têm um prazo para concluir
	srv := config.NovoServidor(r)
	go func() {
		log.Printf("servidor iniciado na porta %s", config.Porta())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("erro ao iniciar servidor: %v", err)
		}
	}()

	<-ctx.Done()

	log.Println("encerrando: aguardando requisições em andamento")
	ctxDesligamento, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	if err := srv.Shutdown(ctxDesligamento); err != nil {
		log.Printf("desligamento forçado: %v", err)
	}

	// espera o worker parar e os emails assíncronos pendentes terminarem
	tarefasEmFundo.Wait()
	log.Println("servidor encerrado")
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
		config.SMTPStartTLS(), config.EmailRemetente(), config.EmailRemetenteNome(),
	)
	if err != nil {
		log.Fatalf("erro ao configurar SMTP: %v", err)
	}
	return m
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
