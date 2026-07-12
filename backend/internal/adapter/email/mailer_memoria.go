package email

import "sync"

// MailerMemoria é um enviador fake que guarda as mensagens em memória, usado
// nos testes para verificar o que seria enviado sem depender de SMTP.
type MailerMemoria struct {
	mu       sync.Mutex
	enviadas []Mensagem
}

// NovaMailerMemoria cria um MailerMemoria vazio.
func NovaMailerMemoria() *MailerMemoria {
	return &MailerMemoria{}
}

func (m *MailerMemoria) Enviar(msg Mensagem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enviadas = append(m.enviadas, msg)
	return nil
}

// Enviadas devolve uma cópia das mensagens enviadas até agora.
func (m *MailerMemoria) Enviadas() []Mensagem {
	m.mu.Lock()
	defer m.mu.Unlock()
	copia := make([]Mensagem, len(m.enviadas))
	copy(copia, m.enviadas)
	return copia
}

// Limpar descarta as mensagens capturadas até agora — útil para isolar, num
// teste, os emails de uma ação específica dos emails de setup anteriores.
func (m *MailerMemoria) Limpar() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enviadas = nil
}
