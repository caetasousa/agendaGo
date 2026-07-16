package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"time"
)

//go:embed templates/*.html
var arquivosTemplates embed.FS

var meses = [...]string{
	"janeiro", "fevereiro", "março", "abril", "maio", "junho",
	"julho", "agosto", "setembro", "outubro", "novembro", "dezembro",
}

// formatarData devolve a data por extenso em português (ex: "12 de julho de 2026").
func formatarData(d time.Time) string {
	return fmt.Sprintf("%d de %s de %d", d.Day(), meses[d.Month()-1], d.Year())
}

// formatarHorario converte minutos desde a meia-noite em "HH:MM".
func formatarHorario(minutos int) string {
	return fmt.Sprintf("%02d:%02d", minutos/60, minutos%60)
}

// Notificador implementa o envio de todos os emails do sistema: recuperação
// de senha e eventos de agendamento. Renderiza os templates HTML e delega o
// transporte ao enviador configurado (SMTP, memória ou nulo).
//
// O envio roda através de executar, o que permite tanto disparo assíncrono
// em produção (goroutine, para não bloquear o request) quanto síncrono nos
// testes (chamada direta, para permitir assert logo após o Executar do use case).
type Notificador struct {
	mailer      enviador
	templates   *template.Template
	urlFrontend string
	fuso        *time.Location
	executar    func(func())
}

// NovoNotificador cria um Notificador, parseando os templates embutidos.
// Falha no boot (panic) se algum template estiver malformado — mais seguro
// que descobrir isso só quando o primeiro email precisar ser enviado.
func NovoNotificador(mailer enviador, urlFrontend string, fuso *time.Location, executar func(func())) *Notificador {
	tmpl, err := template.ParseFS(arquivosTemplates, "templates/*.html")
	if err != nil {
		panic("email: templates inválidos: " + err.Error())
	}
	return &Notificador{
		mailer:      mailer,
		templates:   tmpl,
		urlFrontend: urlFrontend,
		fuso:        fuso,
		executar:    executar,
	}
}

// ExecutorGoroutine devolve um executor que dispara cada envio em uma
// goroutine própria, registrada no WaitGroup para que o desligamento
// gracioso espere os envios pendentes terminarem.
func ExecutorGoroutine(wg waitGroup) func(func()) {
	return func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}
}

// waitGroup é o subconjunto de *sync.WaitGroup usado por ExecutorGoroutine.
type waitGroup interface {
	Add(delta int)
	Done()
}

// ExecutorSincrono roda o envio na mesma goroutine do chamador — usado nos
// testes, para que o assert logo após o Executar do use case já veja o
// email capturado pelo MailerMemoria.
func ExecutorSincrono(fn func()) { fn() }

func (n *Notificador) renderizar(nomeTemplate string, dados any) (string, error) {
	var buf bytes.Buffer
	if err := n.templates.ExecuteTemplate(&buf, nomeTemplate, dados); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (n *Notificador) enviar(destino, nomeDestino, assunto, nomeTemplate string, dados any) {
	n.executar(func() {
		html, err := n.renderizar(nomeTemplate, dados)
		if err != nil {
			slog.Error("email: erro ao renderizar template",
				slog.String("template", nomeTemplate), slog.String("erro", err.Error()))
			return
		}
		msg := Mensagem{Para: destino, NomePara: nomeDestino, Assunto: assunto, HTML: html}
		if err := n.mailer.Enviar(msg); err != nil {
			slog.Error("email: erro ao enviar",
				slog.String("template", nomeTemplate), slog.String("para", destino), slog.String("erro", err.Error()))
		}
	})
}

// EnviarRecuperacaoSenha envia o link de redefinição de senha. Implementa a
// interface enviadorRecuperacao de usecase/auth.
func (n *Notificador) EnviarRecuperacaoSenha(email, nome, token string, expiraEm time.Time) {
	link := fmt.Sprintf("%s/redefinir-senha?token=%s", n.urlFrontend, token)
	dados := struct {
		Nome            string
		Link            string
		ExpiraEmMinutos int
	}{
		Nome:            nome,
		Link:            link,
		ExpiraEmMinutos: int(time.Until(expiraEm).Minutes()),
	}
	n.enviar(email, nome, "Redefinição de senha — agendaGo", "recuperacao_senha.html", dados)
}

// EnviarConfirmacaoCadastro envia o link de confirmação de cadastro. Implementa
// parte da interface enviadorCadastro de usecase/client.
func (n *Notificador) EnviarConfirmacaoCadastro(email, nome, token string, expiraEm time.Time) {
	link := fmt.Sprintf("%s/confirmar-cadastro?token=%s", n.urlFrontend, token)
	dados := struct {
		Nome          string
		Link          string
		ExpiraEmHoras int
	}{
		Nome:          nome,
		Link:          link,
		ExpiraEmHoras: int(time.Until(expiraEm).Hours()),
	}
	n.enviar(email, nome, "Confirme seu cadastro — agendaGo", "confirmacao_cadastro.html", dados)
}

// EnviarAvisoContaExistente avisa que o email já tem conta, no lugar do link de
// confirmação — enviado quando alguém tenta se cadastrar com um email já
// registrado, sem que a resposta HTTP revele isso.
func (n *Notificador) EnviarAvisoContaExistente(email, nome string) {
	link := fmt.Sprintf("%s/login", n.urlFrontend)
	dados := struct {
		Nome            string
		Link            string
		LinkRecuperacao string
	}{
		Nome:            nome,
		Link:            link,
		LinkRecuperacao: n.urlFrontend + "/recuperar-senha",
	}
	n.enviar(email, nome, "Você já tem uma conta — agendaGo", "conta_existente.html", dados)
}
