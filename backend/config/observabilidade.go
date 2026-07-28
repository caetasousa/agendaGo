// Rastreamento de erro: envia panics e logs de nível ERROR para um coletor
// compatível com o protocolo do Sentry (o próprio Sentry ou uma instância
// GlitchTip auto-hospedada). É complementar ao log estruturado, não substituto:
// o log responde "o que aconteceu no servidor"; o coletor responde "qual erro
// está acontecendo, com que frequência, desde quando e em qual linha".
//
// Fica desligado enquanto SENTRY_DSN estiver vazio — mesmo padrão de
// EmailAtivo() e OAuthGoogleAtivo().
package config

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"agendago/internal/pkg/logging"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5/middleware"
)

// timeoutEnvioSentry é quanto o desligamento espera pelo envio dos eventos
// pendentes. Curto de propósito: um coletor fora do ar não pode atrasar o
// shutdown da API.
const timeoutEnvioSentry = 2 * time.Second

// SentryDSN é o endereço do coletor de erros (env SENTRY_DSN). Vazio desliga o
// recurso.
func SentryDSN() string {
	return os.Getenv("SENTRY_DSN")
}

// RastreamentoErroAtivo informa se há coletor de erros configurado.
func RastreamentoErroAtivo() bool {
	return SentryDSN() != ""
}

// IniciarRastreamentoErro liga o coletor de erros e passa a encaminhar todo
// log de nível ERROR para ele. Não faz nada quando SENTRY_DSN está vazio.
// Devolve erro só quando o DSN existe mas é inválido — o que é motivo para
// abortar o boot, já que o operador pediu o recurso explicitamente.
func IniciarRastreamentoErro() error {
	if !RastreamentoErroAtivo() {
		return nil
	}
	ambiente := "desenvolvimento"
	if EhProducao() {
		ambiente = "producao"
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:         SentryDSN(),
		Environment: ambiente,
		// O corpo, os cabeçalhos e a query da requisição são descartados antes
		// do envio: o projeto trata email, telefone e token como dado que não
		// sai em log (ver logging.Rota), e não faria sentido vazá-los para um
		// terceiro justamente no caminho de erro. O contexto útil vai por tag,
		// montada à mão em MiddlewareRastreamentoErro.
		BeforeSend: descartarDadosSensiveis,
	})
	if err != nil {
		return err
	}
	slog.SetDefault(slog.New(handlerSentry{Handler: slog.Default().Handler()}))
	return nil
}

// FecharRastreamentoErro entrega os eventos ainda em fila antes do processo
// morrer. Deve ser chamada no desligamento; sem ela, o erro que derrubou a API
// é justamente o que se perde.
func FecharRastreamentoErro() {
	if RastreamentoErroAtivo() {
		sentry.Flush(timeoutEnvioSentry)
	}
}

// descartarDadosSensiveis zera tudo que possa carregar PII do evento antes do
// envio. É uma lista de negação deliberadamente ampla: na dúvida, não envia.
func descartarDadosSensiveis(evento *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	evento.Request = nil
	evento.User = sentry.User{}
	return evento
}

// MiddlewareRastreamentoErro captura o panic de uma requisição, envia ao
// coletor com rota, método e request_id, e relança — o Recoverer do chi, que
// vem logo depois na cadeia, é quem transforma isso no 500 da resposta.
//
// A rota é lida dentro do recover, e não antes: o padrão casado
// (`/providers/{id}`) só existe depois do roteamento, e usar o caminho bruto
// registraria o token que viaja em rotas como /agendamentos/cancelar/{token}.
func MiddlewareRastreamentoErro(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.CurrentHub().Clone()
		ctx := sentry.SetHubOnContext(r.Context(), hub)
		r = r.WithContext(ctx)

		defer func() {
			p := recover()
			if p == nil {
				return
			}
			hub.Scope().SetTag("rota", logging.Rota(r))
			hub.Scope().SetTag("metodo", r.Method)
			hub.Scope().SetTag("request_id", middleware.GetReqID(ctx))
			hub.RecoverWithContext(ctx, p)
			panic(p)
		}()

		next.ServeHTTP(w, r)
	})
}

// handlerSentry encaminha ao coletor todo registro de nível ERROR, além de
// entregá-lo ao handler original. Interceptar no slog em vez de espalhar
// chamadas pelo código alcança de uma vez os pontos que hoje só logam e seguem
// — falha de envio de email, erro de worker, banco indisponível no /ready.
type handlerSentry struct {
	slog.Handler
}

// Handle envia o registro ao coletor quando o nível é ERROR ou acima.
func (h handlerSentry) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		mensagem := r.Message
		r.Attrs(func(a slog.Attr) bool {
			// Só o atributo "erro" acompanha a mensagem: os demais podem
			// carregar identificador de usuário, e o valor de diagnóstico
			// deles não compensa o risco.
			if a.Key == "erro" {
				mensagem += ": " + a.Value.String()
				return false
			}
			return true
		})
		sentry.CaptureMessage(mensagem)
	}
	return h.Handler.Handle(ctx, r)
}

// WithAttrs preserva o encaminhamento ao coletor no logger derivado.
func (h handlerSentry) WithAttrs(attrs []slog.Attr) slog.Handler {
	return handlerSentry{Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup preserva o encaminhamento ao coletor no logger derivado.
func (h handlerSentry) WithGroup(nome string) slog.Handler {
	return handlerSentry{Handler: h.Handler.WithGroup(nome)}
}
