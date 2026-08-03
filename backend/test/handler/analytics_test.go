package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"agendago/internal/adapter/http/handler"
	"agendago/internal/adapter/http/middleware"
	"agendago/internal/adapter/security"
	"agendago/internal/domain/appointment"
	"agendago/internal/domain/client"
	ucanalytics "agendago/internal/usecase/analytics"
	ucauth "agendago/internal/usecase/auth"
	ucavailability "agendago/internal/usecase/availability"
	"agendago/test/repository/memoria"

	"github.com/go-chi/chi/v5"
)

// novoRouterMetricas monta o router com um prestador que já atendeu na semana
// de 2026-08-10, espelhando o wiring de main.go: a rota vive atrás de
// Autenticar + ExigirProvider.
func novoRouterMetricas(t *testing.T) *chi.Mux {
	t.Helper()
	hasher := security.NovoHasherArgon2id()

	usuarios, membros, providers := fakesDePrestador()
	clientRepo := memoria.NovoClientMemoria()
	sessionRepo := memoria.NovoSessionMemoria()
	availabilityRepo := memoria.NovoAvailabilityMemoria()
	appointmentRepo := memoria.NovoAppointmentMemoria()
	ocupacaoRepo := memoria.NovoOcupacaoMemoria()

	senhaHash, _ := hasher.Gerar("12345678")
	_, p := criarPrestador(usuarios, membros, providers, "provider-1", "João Silva", "joao@email.com", "11999998888", senhaHash)
	p.AtivarAgenda()
	providers.Salvar(p)

	c, _ := client.NovoComConta("client-1", "Maria Silva", "maria@email.com", senhaHash)
	clientRepo.Salvar(c)

	// segunda 2026-08-10, das 9h às 10h, atendimento realizado
	criacao := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	segunda := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	ag, err := appointment.Novo("ag-1", "provider-1", "client-1", segunda, 540, 600, criacao, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("agendamento de teste inválido: %v", err)
	}
	ag.Confirmar(criacao)
	ag.MarcarRealizado(time.Date(2026, 8, 10, 9, 30, 0, 0, time.UTC), time.UTC)
	if err := appointmentRepo.SalvarSeLivre(ag, criacao); err != nil {
		t.Fatalf("salvar agendamento: %v", err)
	}

	loginProvider := ucauth.NovoLoginProviderUseCase(usuarios, membros, providers, sessionRepo, hasher)
	loginClient := ucauth.NovoLoginClientUseCase(clientRepo, sessionRepo, hasher)
	validarSessao := ucauth.NovoValidarSessaoUseCase(sessionRepo, membros)

	identidadeDoContexto := func(req *http.Request) (ucauth.Identidade, bool) {
		return middleware.IdentidadeDoContexto(req.Context())
	}

	consultarAgenda := ucavailability.NovoConsultarAgendaUseCase(availabilityRepo, providers, membros)
	metricas := ucanalytics.NovoMetricasUseCase(consultarAgenda, appointmentRepo, ocupacaoRepo)

	analyticsHandler := handler.NovoAnalyticsHandler(metricas, identidadeDoContexto)
	authHandler := handler.NovoAuthHandler(loginProvider, loginClient, nil, nil, nil, false, nil, identidadeDoContexto)
	authMw := middleware.NovoAuth(validarSessao, false)

	router := chi.NewRouter()
	router.Post("/auth/provider/login", authHandler.LoginProvider)
	router.Post("/auth/client/login", authHandler.LoginClient)
	router.Group(func(router chi.Router) {
		router.Use(authMw.Autenticar)
		router.Use(middleware.ExigirProvider)
		router.Get("/providers/me/metricas", analyticsHandler.Metricas)
	})

	return router
}

func TestHandlerMetricas(t *testing.T) {
	const semana = "/providers/me/metricas?de=2026-08-10&ate=2026-08-16"

	t.Run("GET resume o período do prestador autenticado", func(t *testing.T) {
		r := novoRouterMetricas(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		rr := requisicaoComCookie(t, r, http.MethodGet, semana, nil, cookie)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d, body: %s", rr.Code, rr.Body.String())
		}

		var resp map[string]any
		json.NewDecoder(rr.Body).Decode(&resp)

		if resp["total"].(float64) != 1 {
			t.Errorf("esperava 1 agendamento no total, got: %v", resp["total"])
		}
		porStatus := resp["porStatus"].(map[string]any)
		if len(porStatus) != len(appointment.TodosOsStatus) {
			t.Errorf("esperava o funil completo (%d status), got: %d", len(appointment.TodosOsStatus), len(porStatus))
		}
		if porStatus["REALIZADO"].(float64) != 1 || porStatus["CANCELADO"].(float64) != 0 {
			t.Errorf("esperava 1 realizado e os demais zerados, got: %v", porStatus)
		}
		// 5 dias úteis × 480 minutos de expediente padrão; 60 reservados
		if resp["minutosOfertados"].(float64) != 2400 || resp["minutosReservados"].(float64) != 60 {
			t.Errorf("esperava 60 de 2400 minutos, got: %v de %v", resp["minutosReservados"], resp["minutosOfertados"])
		}
		if resp["taxaComparecimento"].(float64) != 1 {
			t.Errorf("esperava comparecimento de 100%%, got: %v", resp["taxaComparecimento"])
		}
	})

	t.Run("taxa sem base para calcular vem nula, não zero", func(t *testing.T) {
		r := novoRouterMetricas(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		// semana sem nenhum atendimento concluído
		rr := requisicaoComCookie(t, r, http.MethodGet, "/providers/me/metricas?de=2026-09-07&ate=2026-09-13", nil, cookie)

		var resp map[string]any
		json.NewDecoder(rr.Body).Decode(&resp)
		if valor, presente := resp["taxaComparecimento"]; !presente || valor != nil {
			t.Errorf("esperava taxaComparecimento null, got: %v", valor)
		}
	})

	t.Run("GET retorna 400 sem período e para período invertido", func(t *testing.T) {
		r := novoRouterMetricas(t)
		cookie := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		semPeriodo := requisicaoComCookie(t, r, http.MethodGet, "/providers/me/metricas", nil, cookie)
		if semPeriodo.Code != http.StatusBadRequest {
			t.Errorf("esperava 400 sem parâmetros, got: %d", semPeriodo.Code)
		}

		invertido := requisicaoComCookie(t, r, http.MethodGet, "/providers/me/metricas?de=2026-08-16&ate=2026-08-10", nil, cookie)
		if invertido.Code != http.StatusBadRequest {
			t.Errorf("esperava 400 para período invertido, got: %d", invertido.Code)
		}
	})

	t.Run("GET retorna 401 sem cookie e 403 para cliente", func(t *testing.T) {
		r := novoRouterMetricas(t)

		semCookie := requisicaoComCookie(t, r, http.MethodGet, semana, nil, nil)
		if semCookie.Code != http.StatusUnauthorized {
			t.Errorf("esperava 401, got: %d", semCookie.Code)
		}

		cookieCliente := loginEObterCookie(t, r, "/auth/client/login", "maria@email.com", "12345678")
		deCliente := requisicaoComCookie(t, r, http.MethodGet, semana, nil, cookieCliente)
		if deCliente.Code != http.StatusForbidden {
			t.Errorf("esperava 403, got: %d", deCliente.Code)
		}
	})
}
