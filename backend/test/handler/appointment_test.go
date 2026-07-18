package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"agendago/internal/adapter/email"
	"agendago/internal/adapter/http/handler"
	"agendago/internal/adapter/http/middleware"
	"agendago/internal/adapter/repository"
	"agendago/internal/adapter/security"
	"agendago/internal/domain/client"
	"agendago/internal/domain/provider"
	ucappointment "agendago/internal/usecase/appointment"
	ucauth "agendago/internal/usecase/auth"
	ucavailability "agendago/internal/usecase/availability"

	"github.com/go-chi/chi/v5"
)

// novoRouterAgendamento monta um router chi com um prestador ativo e um
// cliente cadastrados e as rotas de agendamento, espelhando o wiring de main.go.
// providerRepo é devolvido para os testes que precisam mutar o prestador
// (ex.: desativar a marcação pelo prestador em preferências) depois de montado o router.
func novoRouterAgendamento(t *testing.T) (r *chi.Mux, providerID string, mailer *email.MailerMemoria, providerRepo *repository.ProviderMemoria) {
	t.Helper()
	hasher := security.NovoHasherArgon2id()

	providerRepo = repository.NovoProviderMemoria()
	clientRepo := repository.NovoClientMemoria()
	sessionRepo := repository.NovoSessionMemoria()
	availabilityRepo := repository.NovoAvailabilityMemoria()
	appointmentRepo := repository.NovoAppointmentMemoria()

	senhaHash, _ := hasher.Gerar("12345678")
	p, _ := provider.Novo("11111111-1111-1111-1111-111111111111", "João Silva", "joao@email.com", "11999998888", senhaHash)
	p.AtivarAgenda()
	providerRepo.Salvar(p)

	c, _ := client.NovoComConta("22222222-2222-2222-2222-222222222222", "Maria Souza", "maria@email.com", senhaHash)
	clientRepo.Salvar(c)

	loginProvider := ucauth.NovoLoginProviderUseCase(providerRepo, sessionRepo, hasher)
	loginClient := ucauth.NovoLoginClientUseCase(clientRepo, sessionRepo, hasher)
	validarSessao := ucauth.NovoValidarSessaoUseCase(sessionRepo)

	identidadeDoContexto := func(req *http.Request) (ucauth.Identidade, bool) {
		return middleware.IdentidadeDoContexto(req.Context())
	}

	mailer = email.NovaMailerMemoria()
	notificador := email.NovoNotificador(mailer, "http://localhost:5173", time.UTC, email.ExecutorSincrono)
	cancelamentoRepo := repository.NovoCancellationMemoria()
	preCadastroRepo := repository.NovoPreCadastroMemoria()
	resolvedor := ucavailability.NovoConsultarDisponibilidadeUseCase(availabilityRepo, providerRepo)
	consultarSlots := ucappointment.NovoConsultarSlotsUseCase(resolvedor, appointmentRepo, providerRepo, time.UTC)
	solicitar := ucappointment.NovoSolicitarUseCase(consultarSlots, appointmentRepo, clientRepo, providerRepo, notificador, 24*time.Hour)
	solicitarConvidado := ucappointment.NovoSolicitarConvidadoUseCase(solicitar, clientRepo, providerRepo, cancelamentoRepo, preCadastroRepo, notificador)
	marcarPeloPrestador := ucappointment.NovoMarcarPeloPrestadorUseCase(solicitar, clientRepo, providerRepo)
	transicionar := ucappointment.NovoTransicionarUseCase(appointmentRepo, providerRepo, clientRepo, cancelamentoRepo, preCadastroRepo, notificador, 24*time.Hour, time.UTC)
	cancelarPorToken := ucappointment.NovoCancelarPorTokenUseCase(appointmentRepo, cancelamentoRepo, providerRepo, clientRepo, notificador, 24*time.Hour, time.UTC)
	listar := ucappointment.NovoListarUseCase(appointmentRepo, providerRepo, clientRepo)

	appointmentHandler := handler.NovoAppointmentHandler(consultarSlots, solicitar, solicitarConvidado, marcarPeloPrestador, transicionar, cancelarPorToken, listar, identidadeDoContexto)
	authHandler := handler.NovoAuthHandler(loginProvider, loginClient, nil, nil, nil, false, identidadeDoContexto)
	authMw := middleware.NovoAuth(validarSessao)

	router := chi.NewRouter()
	router.Post("/auth/provider/login", authHandler.LoginProvider)
	router.Post("/auth/client/login", authHandler.LoginClient)
	router.Get("/providers/{id}/slots", appointmentHandler.ConsultarSlots)
	router.Post("/agendamentos/convidado", appointmentHandler.SolicitarConvidado)
	router.Get("/agendamentos/cancelar/{token}", appointmentHandler.DetalharCancelamento)
	router.Post("/agendamentos/cancelar/{token}", appointmentHandler.CancelarPorToken)
	router.Group(func(router chi.Router) {
		router.Use(authMw.Autenticar)
		router.Use(middleware.ExigirProvider)
		router.Get("/providers/me/agendamentos", appointmentHandler.ListarDoPrestador)
		router.Get("/providers/me/slots", appointmentHandler.ConsultarSlotsDoPrestador)
		router.Post("/providers/me/agendamentos", appointmentHandler.MarcarPeloPrestador)
	})
	router.Group(func(router chi.Router) {
		router.Use(authMw.Autenticar)
		router.Use(middleware.ExigirClient)
		router.Post("/agendamentos", appointmentHandler.Solicitar)
		router.Get("/clients/me/agendamentos", appointmentHandler.ListarDoCliente)
	})
	router.Group(func(router chi.Router) {
		router.Use(authMw.Autenticar)
		router.Post("/agendamentos/{id}/confirmar", appointmentHandler.Confirmar)
		router.Post("/agendamentos/{id}/recusar", appointmentHandler.Recusar)
		router.Post("/agendamentos/{id}/cancelar", appointmentHandler.Cancelar)
		router.Post("/agendamentos/{id}/realizado", appointmentHandler.MarcarRealizado)
		router.Post("/agendamentos/{id}/nao-compareceu", appointmentHandler.MarcarNaoCompareceu)
	})

	return router, p.ID, mailer, providerRepo
}

// dataFutura devolve uma segunda-feira ao menos 30 dias à frente, para os
// testes de handler (que usam time.Now real) nunca caírem em dia passado.
func dataFutura(t *testing.T) string {
	t.Helper()
	d := time.Now().AddDate(0, 0, 30)
	for d.Weekday() != time.Monday {
		d = d.AddDate(0, 0, 1)
	}
	return d.Format("2006-01-02")
}

func TestHandlerAgendamento(t *testing.T) {
	t.Run("fluxo completo: slots públicos, solicitação do cliente e confirmação do prestador", func(t *testing.T) {
		r, providerID, _, _ := novoRouterAgendamento(t)
		data := dataFutura(t)

		// slots são públicos: prestador ativo oferta o expediente padrão fatiado
		rr := requisicaoComCookie(t, r, http.MethodGet,
			fmt.Sprintf("/providers/%s/slots?de=%s&ate=%s", providerID, data, data), nil, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200 nos slots, got: %d, body: %s", rr.Code, rr.Body.String())
		}
		var slots map[string]any
		json.NewDecoder(rr.Body).Decode(&slots)
		dias := slots["dias"].([]any)
		if len(dias[0].(map[string]any)["slots"].([]any)) == 0 {
			t.Fatal("esperava slots ofertados no dia útil")
		}

		// cliente solicita o primeiro slot
		cookieCliente := loginEObterCookie(t, r, "/auth/client/login", "maria@email.com", "12345678")
		corpo := map[string]any{"providerId": providerID, "data": data, "inicioMinutos": 8 * 60}
		rr = requisicaoComCookie(t, r, http.MethodPost, "/agendamentos", corpo, cookieCliente)
		if rr.Code != http.StatusCreated {
			t.Fatalf("esperava 201 na solicitação, got: %d, body: %s", rr.Code, rr.Body.String())
		}
		var criado map[string]any
		json.NewDecoder(rr.Body).Decode(&criado)
		if criado["status"] != "SOLICITADO" {
			t.Errorf("esperava SOLICITADO, got: %v", criado["status"])
		}

		// o slot ocupado sai da oferta
		rr = requisicaoComCookie(t, r, http.MethodGet,
			fmt.Sprintf("/providers/%s/slots?de=%s&ate=%s", providerID, data, data), nil, nil)
		json.NewDecoder(rr.Body).Decode(&slots)
		for _, s := range slots["dias"].([]any)[0].(map[string]any)["slots"].([]any) {
			if s.(map[string]any)["inicioMinutos"].(float64) == 480 {
				t.Error("slot das 08:00 deveria ter saído da oferta")
			}
		}

		// segundo cliente disputando o mesmo horário leva 409/erro de indisponível
		rr = requisicaoComCookie(t, r, http.MethodPost, "/agendamentos", corpo, cookieCliente)
		if rr.Code != http.StatusConflict {
			t.Errorf("esperava 409 na disputa do slot, got: %d", rr.Code)
		}

		// prestador vê a solicitação e confirma
		cookiePrestador := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")
		rr = requisicaoComCookie(t, r, http.MethodGet, "/providers/me/agendamentos", nil, cookiePrestador)
		var lista map[string]any
		json.NewDecoder(rr.Body).Decode(&lista)
		agendamentos := lista["agendamentos"].([]any)
		if len(agendamentos) != 1 {
			t.Fatalf("esperava 1 agendamento na lista do prestador, got: %d", len(agendamentos))
		}
		id := agendamentos[0].(map[string]any)["id"].(string)

		rr = requisicaoComCookie(t, r, http.MethodPost, "/agendamentos/"+id+"/confirmar", nil, cookiePrestador)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("esperava 204 na confirmação, got: %d, body: %s", rr.Code, rr.Body.String())
		}

		// cliente vê o agendamento confirmado e cancela (com folga de antecedência)
		rr = requisicaoComCookie(t, r, http.MethodGet, "/clients/me/agendamentos", nil, cookieCliente)
		json.NewDecoder(rr.Body).Decode(&lista)
		if lista["agendamentos"].([]any)[0].(map[string]any)["status"] != "CONFIRMADO" {
			t.Error("esperava CONFIRMADO na visão do cliente")
		}

		rr = requisicaoComCookie(t, r, http.MethodPost, "/agendamentos/"+id+"/cancelar", nil, cookieCliente)
		if rr.Code != http.StatusNoContent {
			t.Errorf("esperava 204 no cancelamento, got: %d, body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("prestador recusa solicitação; concluir antes da hora leva 409", func(t *testing.T) {
		r, providerID, _, _ := novoRouterAgendamento(t)
		data := dataFutura(t)

		cookieCliente := loginEObterCookie(t, r, "/auth/client/login", "maria@email.com", "12345678")
		corpo := map[string]any{"providerId": providerID, "data": data, "inicioMinutos": 8 * 60}
		rr := requisicaoComCookie(t, r, http.MethodPost, "/agendamentos", corpo, cookieCliente)
		if rr.Code != http.StatusCreated {
			t.Fatalf("esperava 201 na solicitação, got: %d", rr.Code)
		}
		var criado map[string]any
		json.NewDecoder(rr.Body).Decode(&criado)
		id := criado["id"].(string)

		cookiePrestador := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		// concluir uma solicitação pendente (e futura) é transição inválida → 409
		rr = requisicaoComCookie(t, r, http.MethodPost, "/agendamentos/"+id+"/realizado", nil, cookiePrestador)
		if rr.Code != http.StatusConflict {
			t.Errorf("esperava 409 ao marcar realizado antes da hora, got: %d, body: %s", rr.Code, rr.Body.String())
		}
		rr = requisicaoComCookie(t, r, http.MethodPost, "/agendamentos/"+id+"/nao-compareceu", nil, cookiePrestador)
		if rr.Code != http.StatusConflict {
			t.Errorf("esperava 409 ao marcar ausência antes da hora, got: %d, body: %s", rr.Code, rr.Body.String())
		}

		// recusar a pendência funciona
		rr = requisicaoComCookie(t, r, http.MethodPost, "/agendamentos/"+id+"/recusar", nil, cookiePrestador)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("esperava 204 na recusa, got: %d, body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("prestador não solicita agendamento (rota exige cliente)", func(t *testing.T) {
		r, providerID, _, _ := novoRouterAgendamento(t)
		cookiePrestador := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		corpo := map[string]any{"providerId": providerID, "data": dataFutura(t), "inicioMinutos": 480}
		rr := requisicaoComCookie(t, r, http.MethodPost, "/agendamentos", corpo, cookiePrestador)
		if rr.Code != http.StatusForbidden {
			t.Errorf("esperava 403, got: %d", rr.Code)
		}
	})

	t.Run("solicitação sem sessão leva 401 e com corpo inválido 400", func(t *testing.T) {
		r, providerID, _, _ := novoRouterAgendamento(t)

		corpo := map[string]any{"providerId": providerID, "data": dataFutura(t), "inicioMinutos": 480}
		rr := requisicaoComCookie(t, r, http.MethodPost, "/agendamentos", corpo, nil)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("esperava 401, got: %d", rr.Code)
		}

		cookieCliente := loginEObterCookie(t, r, "/auth/client/login", "maria@email.com", "12345678")
		rr = requisicaoComCookie(t, r, http.MethodPost, "/agendamentos",
			map[string]any{"providerId": "não-é-uuid", "data": "amanhã", "inicioMinutos": -1}, cookieCliente)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})

	t.Run("slots de prestador inexistente leva 404 e período inválido 400", func(t *testing.T) {
		r, _, _, _ := novoRouterAgendamento(t)
		data := dataFutura(t)

		rr := requisicaoComCookie(t, r, http.MethodGet,
			"/providers/99999999-9999-9999-9999-999999999999/slots?de="+data+"&ate="+data, nil, nil)
		if rr.Code != http.StatusNotFound {
			t.Errorf("esperava 404, got: %d", rr.Code)
		}

		rr = requisicaoComCookie(t, r, http.MethodGet,
			"/providers/x/slots?de=hoje&ate=amanha", nil, nil)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400, got: %d", rr.Code)
		}
	})
}

func TestHandlerAgendamentoConvidado(t *testing.T) {
	t.Run("convidado agenda sem login e o prestador enxerga nome/email/telefone", func(t *testing.T) {
		r, providerID, _, _ := novoRouterAgendamento(t)
		data := dataFutura(t)

		// sem cookie: a rota de convidado é pública
		corpo := map[string]any{
			"providerId":    providerID,
			"data":          data,
			"inicioMinutos": 8 * 60,
			"nome":          "Convidada Silva",
			"email":         "convidada@email.com",
			"telefone":      "(11) 99999-8888",
		}
		rr := requisicaoComCookie(t, r, http.MethodPost, "/agendamentos/convidado", corpo, nil)
		if rr.Code != http.StatusCreated {
			t.Fatalf("esperava 201 no agendamento de convidado, got: %d, body: %s", rr.Code, rr.Body.String())
		}

		// o prestador vê o contato do convidado
		cookiePrestador := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")
		rr = requisicaoComCookie(t, r, http.MethodGet, "/providers/me/agendamentos", nil, cookiePrestador)
		var lista map[string]any
		json.NewDecoder(rr.Body).Decode(&lista)
		agendamentos := lista["agendamentos"].([]any)
		if len(agendamentos) != 1 {
			t.Fatalf("esperava 1 agendamento na lista do prestador, got: %d", len(agendamentos))
		}
		a := agendamentos[0].(map[string]any)
		if a["nomeCliente"] != "Convidada Silva" || a["emailCliente"] != "convidada@email.com" || a["telefoneCliente"] != "(11) 99999-8888" {
			t.Errorf("esperava contato do convidado visível ao prestador, got: %+v", a)
		}
	})

	t.Run("e-mail de conta registrada leva 409 e orienta a entrar", func(t *testing.T) {
		r, providerID, _, _ := novoRouterAgendamento(t)
		corpo := map[string]any{
			"providerId":    providerID,
			"data":          dataFutura(t),
			"inicioMinutos": 8 * 60,
			"nome":          "Impostora",
			// e-mail da cliente registrada do ambiente de teste
			"email":    "maria@email.com",
			"telefone": "(11) 99999-8888",
		}
		rr := requisicaoComCookie(t, r, http.MethodPost, "/agendamentos/convidado", corpo, nil)
		if rr.Code != http.StatusConflict {
			t.Errorf("esperava 409 para e-mail com conta, got: %d, body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("telefone curto é rejeitado com 400", func(t *testing.T) {
		r, providerID, _, _ := novoRouterAgendamento(t)
		corpo := map[string]any{
			"providerId":    providerID,
			"data":          dataFutura(t),
			"inicioMinutos": 8 * 60,
			"nome":          "Convidada",
			"email":         "convidada@email.com",
			"telefone":      "123",
		}
		rr := requisicaoComCookie(t, r, http.MethodPost, "/agendamentos/convidado", corpo, nil)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400 para telefone curto, got: %d, body: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestCancelamentoPorTokenHandler(t *testing.T) {
	t.Run("detalhar com token inexistente retorna 404", func(t *testing.T) {
		r, _, _, _ := novoRouterAgendamento(t)
		rr := requisicaoComCookie(t, r, http.MethodGet, "/agendamentos/cancelar/token-inexistente", nil, nil)
		if rr.Code != http.StatusNotFound {
			t.Errorf("esperava 404, got: %d, body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("cancelar com token inexistente retorna 404", func(t *testing.T) {
		r, _, _, _ := novoRouterAgendamento(t)
		rr := requisicaoComCookie(t, r, http.MethodPost, "/agendamentos/cancelar/token-inexistente", nil, nil)
		if rr.Code != http.StatusNotFound {
			t.Errorf("esperava 404, got: %d, body: %s", rr.Code, rr.Body.String())
		}
	})
}

// tokenDoLinkNoMailer extrai o token de um link "marcador+TOKEN" presente no
// primeiro email capturado (via MailerMemoria) que o contém.
func tokenDoLinkNoMailer(t *testing.T, mailer *email.MailerMemoria, marcador string) string {
	t.Helper()
	for _, msg := range mailer.Enviadas() {
		i := strings.Index(msg.HTML, marcador)
		if i < 0 {
			continue
		}
		resto := msg.HTML[i+len(marcador):]
		fim := strings.IndexAny(resto, "\"' ")
		if fim < 0 {
			fim = len(resto)
		}
		return resto[:fim]
	}
	t.Fatalf("nenhum email com link %q foi enviado", marcador)
	return ""
}

func tokenDoLinkCancelamento(t *testing.T, mailer *email.MailerMemoria) string {
	t.Helper()
	return tokenDoLinkNoMailer(t, mailer, "/cancelar-agendamento/")
}

func TestSolicitarConvidadoEnviaLinkDeCadastro(t *testing.T) {
	t.Run("solicitação de convidado já traz o link de cadastro pré-preenchido, sem endpoint extra", func(t *testing.T) {
		r, providerID, mailer, _ := novoRouterAgendamento(t)
		corpo := map[string]any{
			"providerId":    providerID,
			"data":          dataFutura(t),
			"inicioMinutos": 8 * 60,
			"nome":          "Convidada Silva",
			"email":         "convidada-pre@email.com",
			"telefone":      "(11) 99999-8888",
		}
		rr := requisicaoComCookie(t, r, http.MethodPost, "/agendamentos/convidado", corpo, nil)
		if rr.Code != http.StatusCreated {
			t.Fatalf("esperava 201 na solicitação, got: %d, body: %s", rr.Code, rr.Body.String())
		}

		tokenPreCadastro := tokenDoLinkNoMailer(t, mailer, "/cadastro?pre=")
		if tokenPreCadastro == "" {
			t.Fatal("esperava um token de pré-cadastro não vazio")
		}
	})
}

func TestHandlerMarcarPeloPrestador(t *testing.T) {
	t.Run("prestador marca para cliente só com nome e observação, e o slot sai da oferta pública", func(t *testing.T) {
		r, providerID, mailer, _ := novoRouterAgendamento(t)
		data := dataFutura(t)

		cookiePrestador := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")
		corpo := map[string]any{
			"data":          data,
			"inicioMinutos": 8 * 60,
			"nome":          "Cliente Telefone",
			"observacao":    "ligou pedindo horário de manhã",
		}
		rr := requisicaoComCookie(t, r, http.MethodPost, "/providers/me/agendamentos", corpo, cookiePrestador)
		if rr.Code != http.StatusCreated {
			t.Fatalf("esperava 201 na marcação, got: %d, body: %s", rr.Code, rr.Body.String())
		}
		var criado map[string]any
		json.NewDecoder(rr.Body).Decode(&criado)
		if criado["status"] != "CONFIRMADO" {
			t.Errorf("esperava CONFIRMADO, got: %v", criado["status"])
		}
		if criado["marcadoPeloPrestador"] != true {
			t.Errorf("esperava marcadoPeloPrestador=true, got: %v", criado["marcadoPeloPrestador"])
		}
		if criado["observacao"] != "ligou pedindo horário de manhã" {
			t.Errorf("esperava observação na resposta, got: %v", criado["observacao"])
		}

		// marcação pelo prestador não notifica ninguém
		if enviados := mailer.Enviadas(); len(enviados) != 0 {
			t.Errorf("esperava zero emails na marcação pelo prestador, got: %d", len(enviados))
		}

		// o slot marcado sai da oferta pública
		rr = requisicaoComCookie(t, r, http.MethodGet,
			fmt.Sprintf("/providers/%s/slots?de=%s&ate=%s", providerID, data, data), nil, nil)
		var slots map[string]any
		json.NewDecoder(rr.Body).Decode(&slots)
		for _, s := range slots["dias"].([]any)[0].(map[string]any)["slots"].([]any) {
			if s.(map[string]any)["inicioMinutos"].(float64) == 480 {
				t.Error("slot das 08:00 deveria ter saído da oferta pública")
			}
		}

		// e o nome e a observação aparecem na lista do prestador, já CONFIRMADO
		rr = requisicaoComCookie(t, r, http.MethodGet, "/providers/me/agendamentos", nil, cookiePrestador)
		var lista map[string]any
		json.NewDecoder(rr.Body).Decode(&lista)
		a := lista["agendamentos"].([]any)[0].(map[string]any)
		if a["nomeCliente"] != "Cliente Telefone" || a["observacao"] != "ligou pedindo horário de manhã" {
			t.Errorf("esperava nome e observação na lista, got: %+v", a)
		}
		if a["status"] != "CONFIRMADO" || a["marcadoPeloPrestador"] != true {
			t.Errorf("esperava CONFIRMADO e marcadoPeloPrestador=true na lista, got: %+v", a)
		}

		// e o prestador cancela na hora, sem antecedência
		id := a["id"].(string)
		rr = requisicaoComCookie(t, r, http.MethodPost, "/agendamentos/"+id+"/cancelar", nil, cookiePrestador)
		if rr.Code != http.StatusNoContent {
			t.Errorf("esperava 204 no cancelamento sem antecedência, got: %d, body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("prestador com a funcionalidade desativada em preferências leva 403", func(t *testing.T) {
		r, _, _, providerRepo := novoRouterAgendamento(t)
		p, _ := providerRepo.BuscarPorID("11111111-1111-1111-1111-111111111111")
		p.DesativarMarcacaoPeloPrestador()
		providerRepo.Salvar(p)

		cookiePrestador := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")
		corpo := map[string]any{
			"data":          dataFutura(t),
			"inicioMinutos": 8 * 60,
			"nome":          "Cliente Telefone",
		}
		rr := requisicaoComCookie(t, r, http.MethodPost, "/providers/me/agendamentos", corpo, cookiePrestador)
		if rr.Code != http.StatusForbidden {
			t.Errorf("esperava 403 com a funcionalidade desativada, got: %d, body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("nome vazio é rejeitado com 400", func(t *testing.T) {
		r, _, _, _ := novoRouterAgendamento(t)
		cookiePrestador := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")
		corpo := map[string]any{
			"data":          dataFutura(t),
			"inicioMinutos": 8 * 60,
			"nome":          "",
		}
		rr := requisicaoComCookie(t, r, http.MethodPost, "/providers/me/agendamentos", corpo, cookiePrestador)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("esperava 400 para nome vazio, got: %d, body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("cliente autenticado não acessa a rota do prestador (403)", func(t *testing.T) {
		r, _, _, _ := novoRouterAgendamento(t)
		cookieCliente := loginEObterCookie(t, r, "/auth/client/login", "maria@email.com", "12345678")
		corpo := map[string]any{
			"data":          dataFutura(t),
			"inicioMinutos": 8 * 60,
			"nome":          "Cliente Telefone",
		}
		rr := requisicaoComCookie(t, r, http.MethodPost, "/providers/me/agendamentos", corpo, cookieCliente)
		if rr.Code != http.StatusForbidden {
			t.Errorf("esperava 403 para cliente, got: %d", rr.Code)
		}

		rr = requisicaoComCookie(t, r, http.MethodGet,
			fmt.Sprintf("/providers/me/slots?de=%s&ate=%s", dataFutura(t), dataFutura(t)), nil, cookieCliente)
		if rr.Code != http.StatusForbidden {
			t.Errorf("esperava 403 nos slots do prestador para cliente, got: %d", rr.Code)
		}
	})

	t.Run("prestador consulta os próprios slots", func(t *testing.T) {
		r, _, _, _ := novoRouterAgendamento(t)
		data := dataFutura(t)
		cookiePrestador := loginEObterCookie(t, r, "/auth/provider/login", "joao@email.com", "12345678")

		rr := requisicaoComCookie(t, r, http.MethodGet,
			fmt.Sprintf("/providers/me/slots?de=%s&ate=%s", data, data), nil, cookiePrestador)
		if rr.Code != http.StatusOK {
			t.Fatalf("esperava 200, got: %d, body: %s", rr.Code, rr.Body.String())
		}
		var slots map[string]any
		json.NewDecoder(rr.Body).Decode(&slots)
		if len(slots["dias"].([]any)[0].(map[string]any)["slots"].([]any)) == 0 {
			t.Error("esperava slots ofertados ao próprio prestador")
		}
	})
}
