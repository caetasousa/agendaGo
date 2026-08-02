package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/domain/appointment"
	"agendago/internal/domain/availability"
	domocupacao "agendago/internal/domain/ocupacao"
	ucappointment "agendago/internal/usecase/appointment"
	ucavailability "agendago/internal/usecase/availability"
	ucocupacao "agendago/internal/usecase/ocupacao"
	"agendago/test/repository/memoria"
)

// cenárioOcupacao monta um prestador com expediente das 08h às 12h, duração de
// 60 min e sem descanso — quatro slots por dia: 08, 09, 10 e 11.
func cenarioOcupacao(t *testing.T) (*ucocupacao.CriarUseCase, *ucocupacao.ListarUseCase, *ucocupacao.RemoverUseCase, *ucappointment.ConsultarSlotsUseCase, *memoria.AppointmentMemoria, string) {
	t.Helper()

	usuarios, membros, providers := fakesDePrestador()
	_, p := criarPrestador(usuarios, membros, providers, "provider-1", "João", "joao@email.com", "11999998888", "hash")
	p.DefinirDuracaoAtendimento(60)
	p.DefinirDescanso(0)
	bloco, _ := availability.NovoTimeBlock(8*60, 12*60)
	p.DefinirHorariosPadrao([]availability.TimeBlock{bloco})
	p.AtivarAgenda()
	providers.Salvar(p)

	availabilityRepo := memoria.NovoAvailabilityMemoria()
	appointments := memoria.NovoAppointmentMemoria()
	ocupacoes := memoria.NovoOcupacaoMemoria()

	resolvedor := ucavailability.NovoConsultarDisponibilidadeUseCase(availabilityRepo, providers, membros)
	consultarSlots := ucappointment.NovoConsultarSlotsUseCase(resolvedor, appointments, providers, membros, ocupacoes, time.UTC)

	return ucocupacao.NovoCriarUseCase(ocupacoes, appointments),
		ucocupacao.NovoListarUseCase(ocupacoes),
		ucocupacao.NovoRemoverUseCase(ocupacoes),
		consultarSlots, appointments, p.ID
}

func TestOcupacaoRemoveDaOferta(t *testing.T) {
	criar, _, remover, consultarSlots, _, providerID := cenarioOcupacao(t)

	// Uma segunda-feira bem no futuro, para nenhum slot cair no passado.
	data := time.Date(2027, 3, 15, 0, 0, 0, 0, time.UTC)
	agora := time.Date(2027, 3, 1, 8, 0, 0, 0, time.UTC)

	slotsDe := func(t *testing.T) int {
		t.Helper()
		out, err := consultarSlots.Executar(ucappointment.ConsultarSlotsInput{
			ProviderID: providerID, De: data, Ate: data, Agora: agora,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.Dias) != 1 {
			t.Fatalf("esperava 1 dia, got: %d", len(out.Dias))
		}
		return len(out.Dias[0].Slots)
	}

	antes := slotsDe(t)
	if antes != 4 {
		t.Fatalf("esperava 4 slots (08,09,10,11), got: %d", antes)
	}

	o, err := criar.Executar(ucocupacao.CriarInput{
		ProviderID: providerID, Data: data,
		InicioMinutos: 9 * 60, FimMinutos: 10 * 60,
		Titulo: "Médico", Agora: agora,
	})
	if err != nil {
		t.Fatalf("esperava sucesso ao criar compromisso, got: %v", err)
	}

	if depois := slotsDe(t); depois != antes-1 {
		t.Errorf("esperava %d slots com o compromisso das 09h, got: %d", antes-1, depois)
	}

	// Remover devolve o horário à oferta: é o que garante que a ocupação não
	// deixou rastro no cálculo.
	if err := remover.Executar(ucocupacao.RemoverInput{ID: o.ID, ProviderID: providerID}); err != nil {
		t.Fatalf("esperava sucesso ao remover, got: %v", err)
	}
	if voltou := slotsDe(t); voltou != antes {
		t.Errorf("esperava os %d slots de volta após remover, got: %d", antes, voltou)
	}
}

func TestOcupacaoRecusaSobreAgendamento(t *testing.T) {
	criar, _, _, _, appointments, providerID := cenarioOcupacao(t)

	data := time.Date(2027, 3, 15, 0, 0, 0, 0, time.UTC)
	agora := time.Date(2027, 3, 1, 8, 0, 0, 0, time.UTC)

	// Cliente marcado das 09h às 10h.
	a, err := appointment.Novo("ag-1", providerID, "client-1", data, 9*60, 10*60, agora, 24*time.Hour)
	if err != nil {
		t.Fatalf("preparar agendamento: %v", err)
	}
	if err := appointments.SalvarSeLivre(a, agora); err != nil {
		t.Fatalf("persistir agendamento: %v", err)
	}

	_, err = criar.Executar(ucocupacao.CriarInput{
		ProviderID: providerID, Data: data,
		InicioMinutos: 9 * 60, FimMinutos: 10 * 60, Agora: agora,
	})
	if err != domocupacao.ErrConflitoComAgendamento {
		t.Errorf("esperava ErrConflitoComAgendamento, got: %v", err)
	}

	t.Run("horário livre no mesmo dia continua aceitando", func(t *testing.T) {
		_, err := criar.Executar(ucocupacao.CriarInput{
			ProviderID: providerID, Data: data,
			InicioMinutos: 11 * 60, FimMinutos: 12 * 60, Agora: agora,
		})
		if err != nil {
			t.Errorf("esperava sucesso em horário livre, got: %v", err)
		}
	})
}

func TestRemoverOcupacaoDeOutraAgenda(t *testing.T) {
	criar, _, remover, _, _, providerID := cenarioOcupacao(t)

	data := time.Date(2027, 3, 15, 0, 0, 0, 0, time.UTC)
	agora := time.Date(2027, 3, 1, 8, 0, 0, 0, time.UTC)

	o, err := criar.Executar(ucocupacao.CriarInput{
		ProviderID: providerID, Data: data,
		InicioMinutos: 9 * 60, FimMinutos: 10 * 60, Agora: agora,
	})
	if err != nil {
		t.Fatalf("preparar compromisso: %v", err)
	}

	// Mesma resposta de "não encontrado": quem não é dono não descobre que o id
	// existe na agenda de outra pessoa.
	err = remover.Executar(ucocupacao.RemoverInput{ID: o.ID, ProviderID: "outro-provider"})
	if err != ucocupacao.ErrOcupacaoNaoEncontrada {
		t.Errorf("esperava ErrOcupacaoNaoEncontrada, got: %v", err)
	}
}
