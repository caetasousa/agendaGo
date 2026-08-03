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
	"agendago/internal/adapter/email"
	"agendago/internal/adapter/http/handler"
	"agendago/internal/adapter/http/middleware"
	"agendago/internal/adapter/oauth"
	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	"agendago/internal/adapter/worker"
	"agendago/internal/pkg/logging"
	ucadmin "agendago/internal/usecase/admin"
	ucanalytics "agendago/internal/usecase/analytics"
	ucappointment "agendago/internal/usecase/appointment"
	ucauth "agendago/internal/usecase/auth"
	ucavailability "agendago/internal/usecase/availability"
	ucclient "agendago/internal/usecase/client"
	uclgpd "agendago/internal/usecase/lgpd"
	ucmembro "agendago/internal/usecase/membro"
	ucocupacao "agendago/internal/usecase/ocupacao"
	ucprovider "agendago/internal/usecase/provider"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"
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
	usuarioRepo := repository.NovoUsuarioPostgres(pool)
	membroRepo := repository.NovoMembroPostgres(pool)
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
	conviteRepo := repository.NovoConvitePostgres(pool)
	ocupacaoRepo := repository.NovoOcupacaoPostgres(pool)
	auditoriaRepo := repository.NovoAuditoriaPostgres(pool)

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
	solicitarCadastroProvider := ucprovider.NovoSolicitarCadastroUseCase(usuarioRepo, clientRepo, adminRepo, signupRepo, notificador, hasher)
	confirmarCadastroProvider := ucprovider.NovoConfirmarCadastroUseCase(providerRepo, usuarioRepo, membroRepo, clientRepo, adminRepo, signupRepo)
	atualizarPreferencias := ucprovider.NovoAtualizarPreferenciasUseCase(providerRepo, usuarioRepo, membroRepo, conviteRepo)
	solicitarCadastroClient := ucclient.NovoSolicitarCadastroUseCase(clientRepo, usuarioRepo, adminRepo, signupRepo, notificador, hasher)
	confirmarCadastroClient := ucclient.NovoConfirmarCadastroUseCase(clientRepo, usuarioRepo, adminRepo, signupRepo)
	consultarPreCadastro := ucclient.NovoConsultarPreCadastroUseCase(preCadastroRepo)
	concluirPreCadastro := ucclient.NovoConcluirPreCadastroUseCase(clientRepo, usuarioRepo, adminRepo, preCadastroRepo, hasher)
	loginProvider := ucauth.NovoLoginProviderUseCase(usuarioRepo, membroRepo, providerRepo, sessionRepo, hasher)
	loginClient := ucauth.NovoLoginClientUseCase(clientRepo, sessionRepo, hasher)
	loginAdmin := ucauth.NovoLoginAdminUseCase(adminRepo, sessionRepo, hasher)
	loginSocial := novoLoginSocialUseCase(context.Background(), providerRepo, usuarioRepo, membroRepo, clientRepo, adminRepo, socialIdentityRepo, oauthStateRepo, sessionRepo, hasher)
	logout := ucauth.NovoLogoutUseCase(sessionRepo)
	validarSessao := ucauth.NovoValidarSessaoUseCase(sessionRepo, membroRepo)
	perfil := ucauth.NovoPerfilUseCase(usuarioRepo, providerRepo, clientRepo, adminRepo)
	solicitarRecuperacao := ucauth.NovoSolicitarRecuperacaoUseCase(usuarioRepo, membroRepo, providerRepo, clientRepo, passwordResetRepo, notificador)
	redefinirSenha := ucauth.NovoRedefinirSenhaUseCase(usuarioRepo, clientRepo, passwordResetRepo, sessionRepo, hasher)
	convidarMembro := ucmembro.NovoConvidarUseCase(conviteRepo, usuarioRepo, clientRepo, adminRepo, providerRepo, notificador)
	cancelarConvite := ucmembro.NovoCancelarConviteUseCase(conviteRepo)
	consultarConvite := ucmembro.NovoConsultarConviteUseCase(conviteRepo, providerRepo)
	aceitarConvite := ucmembro.NovoAceitarConviteUseCase(conviteRepo, usuarioRepo, membroRepo, providerRepo, hasher)
	listarEquipe := ucmembro.NovoListarEquipeUseCase(membroRepo, usuarioRepo, conviteRepo, providerRepo)
	removerMembro := ucmembro.NovoRemoverMembroUseCase(membroRepo, membroRepo, usuarioRepo, sessionRepo)
	criarOcupacao := ucocupacao.NovoCriarUseCase(ocupacaoRepo, appointmentRepo)
	listarOcupacoes := ucocupacao.NovoListarUseCase(ocupacaoRepo)
	removerOcupacao := ucocupacao.NovoRemoverUseCase(ocupacaoRepo)
	exportarDados := uclgpd.NovoExportarUseCase(clientRepo, appointmentRepo, auditoriaRepo)
	anonimizarCliente := uclgpd.NovoAnonimizarUseCase(clientRepo, sessionRepo, passwordResetRepo, socialIdentityRepo, auditoriaRepo)
	moderar := ucadmin.NovoModerarUseCase(providerRepo, usuarioRepo, membroRepo, clientRepo, sessionRepo, auditoriaRepo)
	consultarAgenda := ucavailability.NovoConsultarAgendaUseCase(availabilityRepo, providerRepo, membroRepo)
	definirDia := ucavailability.NovoDefinirDiaUseCase(availabilityRepo)
	removerDia := ucavailability.NovoRemoverDiaUseCase(availabilityRepo)
	consultarDisponibilidade := ucavailability.NovoConsultarDisponibilidadeUseCase(availabilityRepo, providerRepo, membroRepo)
	listarPrestadores := ucprovider.NovoListarUseCase(providerRepo)
	buscarPrestador := ucprovider.NovoBuscarResumoUseCase(providerRepo, membroRepo)
	consultarSlots := ucappointment.NovoConsultarSlotsUseCase(consultarDisponibilidade, appointmentRepo, providerRepo, membroRepo, ocupacaoRepo, config.FusoHorario)
	solicitarAgendamento := ucappointment.NovoSolicitarUseCase(consultarSlots, appointmentRepo, clientRepo, providerRepo, membroRepo, notificador, config.TTLSolicitacao)
	solicitarConvidado := ucappointment.NovoSolicitarConvidadoUseCase(solicitarAgendamento, clientRepo, providerRepo, membroRepo, cancelamentoRepo, preCadastroRepo, notificador)
	marcarPeloPrestador := ucappointment.NovoMarcarPeloPrestadorUseCase(solicitarAgendamento, clientRepo, providerRepo)
	transicionarAgendamento := ucappointment.NovoTransicionarUseCase(appointmentRepo, providerRepo, membroRepo, clientRepo, cancelamentoRepo, preCadastroRepo, notificador, config.AntecedenciaMinimaCancelamento, config.FusoHorario)
	cancelarPorToken := ucappointment.NovoCancelarPorTokenUseCase(appointmentRepo, cancelamentoRepo, providerRepo, membroRepo, clientRepo, notificador, config.AntecedenciaMinimaCancelamento, config.FusoHorario)
	listarAgendamentos := ucappointment.NovoListarUseCase(appointmentRepo, providerRepo, clientRepo)
	detalharUsuario := ucadmin.NovoDetalharUseCase(providerRepo, usuarioRepo, membroRepo, clientRepo, listarAgendamentos)
	lembrarAgendamento := ucappointment.NovoLembrarUseCase(appointmentRepo, providerRepo, membroRepo, clientRepo, notificador, config.FusoHorario, config.AntecedenciaLembrete)
	metricasDoPrestador := ucanalytics.NovoMetricasUseCase(consultarAgenda, appointmentRepo, ocupacaoRepo)

	// handlers
	identidadeDoContexto := func(r *http.Request) (ucauth.Identidade, bool) {
		return middleware.IdentidadeDoContexto(r.Context())
	}
	providerHandler := handler.NovoProviderHandler(solicitarCadastroProvider, confirmarCadastroProvider, atualizarPreferencias, listarPrestadores, buscarPrestador, identidadeDoContexto)
	membroHandler := handler.NovoMembroHandler(convidarMembro, cancelarConvite, consultarConvite, aceitarConvite, listarEquipe, removerMembro, identidadeDoContexto)
	ocupacaoHandler := handler.NovoOcupacaoHandler(criarOcupacao, listarOcupacoes, removerOcupacao, identidadeDoContexto)
	lgpdHandler := handler.NovoLgpdHandler(exportarDados, anonimizarCliente, config.CookieSeguro(), identidadeDoContexto)
	clientHandler := handler.NovoClientHandler(solicitarCadastroClient, confirmarCadastroClient, consultarPreCadastro, concluirPreCadastro)
	// teto de tentativas por conta, compartilhado entre login e recuperação de
	// senha: o mesmo contador, com chaves de prefixo diferente
	limitadorPorConta := handler.NovoLimitadorPorConta(config.RateLimitLoginPorConta(), config.JanelaLimitePorConta)
	authHandler := handler.NovoAuthHandler(loginProvider, loginClient, loginAdmin, logout, perfil, config.CookieSeguro(), limitadorPorConta, identidadeDoContexto)
	oauthHandler := handler.NovoOAuthHandler(loginSocial, config.CookieSeguro(), config.OrigemFrontend())
	passwordResetHandler := handler.NovoPasswordResetHandler(solicitarRecuperacao, redefinirSenha, limitadorPorConta)
	availabilityHandler := handler.NovoAvailabilityHandler(consultarAgenda, definirDia, removerDia, identidadeDoContexto)
	appointmentHandler := handler.NovoAppointmentHandler(consultarSlots, solicitarAgendamento, solicitarConvidado, marcarPeloPrestador, transicionarAgendamento, cancelarPorToken, listarAgendamentos, identidadeDoContexto)
	adminHandler := handler.NovoAdminHandler(moderar, detalharUsuario, identidadeDoContexto)
	analyticsHandler := handler.NovoAnalyticsHandler(metricasDoPrestador, identidadeDoContexto)

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
	r.Get("/ready", ready(pool))
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
		// Convite: público porque quem foi convidado ainda não tem conta — é
		// justamente o aceite que a cria. O token de uso único é a proteção, e
		// o teto por IP deste grupo cobre a força bruta.
		r.Get("/membros/convite", membroHandler.ConsultarConvite)
		r.Post("/membros/aceitar-convite", membroHandler.AceitarConvite)
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
		// Todas as rotas daqui operam a AGENDA, e por isso passam pelo papel do
		// usuário nela. Hoje os dois papéis podem, então nada muda — o ganho é
		// que um papel novo e mais restrito nasce sem acesso, em vez de nascer
		// com acesso a tudo e depender de alguém lembrar de barrá-lo.
		r.Use(middleware.ExigirGestaoDaAgenda)
		r.Put("/providers/me/preferencias", providerHandler.AtualizarPreferencias)
		r.Get("/providers/me/agenda", availabilityHandler.ConsultarAgenda)
		r.Put("/providers/me/dias/{data}", availabilityHandler.DefinirDia)
		r.Delete("/providers/me/dias/{data}", availabilityHandler.RemoverDia)
		r.Get("/providers/me/agendamentos", appointmentHandler.ListarDoPrestador)
		// resumo analítico do período: funil de status e ocupação do expediente
		r.Get("/providers/me/metricas", analyticsHandler.Metricas)
		// marcação feita pelo próprio prestador (cliente ligou): slots da
		// própria agenda (mesmo fechada ao público) e o registro da reserva
		r.Get("/providers/me/slots", appointmentHandler.ConsultarSlotsDoPrestador)
		r.Post("/providers/me/agendamentos", appointmentHandler.MarcarPeloPrestador)
		// compromisso pessoal: reserva um intervalo do dia para o prestador,
		// tirando-o da oferta sem redefinir o expediente
		r.Get("/providers/me/ocupacoes", ocupacaoHandler.Listar)
		r.Post("/providers/me/ocupacoes", ocupacaoHandler.Criar)
		r.Delete("/providers/me/ocupacoes/{id}", ocupacaoHandler.Remover)
		// Ver quem tem acesso é parte de operar a agenda; mudar quem tem acesso
		// não é — por isso o POST e os DELETE ficam no grupo do dono, abaixo.
		r.Get("/providers/me/membros", membroHandler.ListarEquipe)
	})
	// Administração da equipe: só o dono. É a primeira rota a exigir isso —
	// mudar quem tem acesso à agenda é decisão de quem responde por ela, não de
	// quem apenas a opera no dia a dia.
	r.Group(func(r chi.Router) {
		r.Use(middleware.SemCache)
		r.Use(authMw.Autenticar)
		limitarPorSessao(r, config.RateLimitAutenticadoPorMinuto())
		r.Use(middleware.ExigirProvider)
		r.Use(middleware.ExigirAdministracaoDaConta)
		r.Post("/providers/me/membros", membroHandler.Convidar)
		r.Delete("/providers/me/membros/{id}", membroHandler.Remover)
		r.Delete("/providers/me/convites/{email}", membroHandler.CancelarConvite)
	})
	r.Group(func(r chi.Router) {
		r.Use(middleware.SemCache)
		r.Use(authMw.Autenticar)
		limitarPorSessao(r, config.RateLimitAutenticadoPorMinuto())
		r.Use(middleware.ExigirClient)
		r.Post("/agendamentos", appointmentHandler.Solicitar)
		r.Get("/clients/me/agendamentos", appointmentHandler.ListarDoCliente)
		// LGPD: portabilidade e exclusão dos próprios dados. A exclusão
		// anonimiza em vez de apagar — os agendamentos são da agenda do
		// prestador também, ver internal/usecase/lgpd.
		r.Get("/clients/me/dados", lgpdHandler.ExportarDados)
		r.Delete("/clients/me", lgpdHandler.RemoverConta)
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
	usuarioRepo *repository.UsuarioPostgres,
	membroRepo *repository.MembroPostgres,
	clientRepo *repository.ClientPostgres,
	adminRepo *repository.AdminPostgres,
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

	return ucauth.NovoLoginSocialUseCase(
		google,
		clientRepo, usuarioRepo, membroRepo, providerRepo, adminRepo,
		clientRepo, usuarioRepo, providerRepo, membroRepo,
		socialIdentityRepo, oauthStateRepo, sessionRepo, hasher,
	)
}

// health godoc
//
//	@Summary		Liveness check
//	@Description	Informa que o processo está no ar. Não toca em dependência nenhuma — use /ready para saber se a API consegue atender.
//	@Tags			infra
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Router			/health [get]
func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// timeoutReady limita o ping de readiness: sem teto, um banco travado (em vez de
// fora do ar) seguraria a checagem até o timeout do cliente, e o orquestrador
// ficaria sem resposta justamente quando ela mais importa.
const timeoutReady = 2 * time.Second

// ready godoc
//
//	@Summary		Readiness check
//	@Description	Informa se a API consegue atender de fato: faz ping no pool do banco. Responde 503 quando o banco está indisponível.
//	@Tags			infra
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Failure		503	{object}	map[string]string
//	@Router			/ready [get]
func ready(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), timeoutReady)
		defer cancelar()

		w.Header().Set("Content-Type", "application/json")
		if err := pool.Ping(ctx); err != nil {
			slog.Error("readiness: banco indisponível", slog.String("erro", err.Error()))
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "degradado", "erro": "banco indisponível"})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}
