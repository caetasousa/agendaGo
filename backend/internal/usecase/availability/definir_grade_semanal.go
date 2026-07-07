package availability

import "agendago/internal/domain/availability"

// BlocoInput representa um bloco de horário em minutos, ainda não validado pelo domínio.
type BlocoInput struct {
	InicioMinutos int
	FimMinutos    int
}

// DefinirGradeSemanalInput contém a grade completa a substituir. ProviderID
// vem da identidade da sessão autenticada, nunca do corpo da requisição.
type DefinirGradeSemanalInput struct {
	ProviderID   string
	BlocosPorDia map[availability.DiaSemana][]BlocoInput
}

// DefinirGradeSemanalOutput contém a grade após a substituição, já normalizada
// (blocos adjacentes mesclados).
type DefinirGradeSemanalOutput struct {
	Dias map[availability.DiaSemana][]availability.TimeBlock
}

// DefinirGradeSemanalUseCase substitui a grade semanal inteira do prestador.
type DefinirGradeSemanalUseCase struct {
	repo repositorioWeeklySchedule
}

// NovoDefinirGradeSemanalUseCase cria uma instância de DefinirGradeSemanalUseCase com o repositório injetado.
func NovoDefinirGradeSemanalUseCase(repo repositorioWeeklySchedule) *DefinirGradeSemanalUseCase {
	return &DefinirGradeSemanalUseCase{repo: repo}
}

// Executar valida cada bloco individualmente, monta a grade (validando
// ausência de sobreposição e mesclando blocos adjacentes) e a persiste,
// substituindo por completo a grade anterior do prestador.
func (uc *DefinirGradeSemanalUseCase) Executar(in DefinirGradeSemanalInput) (*DefinirGradeSemanalOutput, error) {
	blocosPorDia := make(map[availability.DiaSemana][]availability.TimeBlock, len(in.BlocosPorDia))
	for dia, blocos := range in.BlocosPorDia {
		for _, b := range blocos {
			bloco, err := availability.NovoTimeBlock(b.InicioMinutos, b.FimMinutos)
			if err != nil {
				return nil, err
			}
			blocosPorDia[dia] = append(blocosPorDia[dia], bloco)
		}
	}

	schedule, err := availability.NovaWeeklySchedule(in.ProviderID, blocosPorDia)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Salvar(schedule); err != nil {
		return nil, err
	}

	return &DefinirGradeSemanalOutput{Dias: schedule.Dias}, nil
}
