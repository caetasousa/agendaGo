package availability

import "agendago/internal/domain/availability"

// ConsultarGradeSemanalInput identifica o prestador cuja grade será consultada.
type ConsultarGradeSemanalInput struct {
	ProviderID string
}

// ConsultarGradeSemanalOutput contém a grade crua como configurada pelo
// prestador (sem fallback comercial — isso é resolvido só por
// ConsultarDisponibilidadeUseCase).
type ConsultarGradeSemanalOutput struct {
	Dias map[availability.DiaSemana][]availability.TimeBlock
}

// ConsultarGradeSemanalUseCase carrega a grade semanal atual do prestador,
// para popular o formulário de edição.
type ConsultarGradeSemanalUseCase struct {
	repo repositorioWeeklySchedule
}

// NovoConsultarGradeSemanalUseCase cria uma instância de ConsultarGradeSemanalUseCase com o repositório injetado.
func NovoConsultarGradeSemanalUseCase(repo repositorioWeeklySchedule) *ConsultarGradeSemanalUseCase {
	return &ConsultarGradeSemanalUseCase{repo: repo}
}

// Executar devolve a grade do prestador. Se ele nunca configurou nada,
// devolve Dias vazio (não o default comercial).
func (uc *ConsultarGradeSemanalUseCase) Executar(in ConsultarGradeSemanalInput) (*ConsultarGradeSemanalOutput, error) {
	schedule, err := uc.repo.Buscar(in.ProviderID)
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		return &ConsultarGradeSemanalOutput{Dias: map[availability.DiaSemana][]availability.TimeBlock{}}, nil
	}
	return &ConsultarGradeSemanalOutput{Dias: schedule.Dias}, nil
}
