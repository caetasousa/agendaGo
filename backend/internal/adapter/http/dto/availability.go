package dto

type BlocoDTO struct {
	InicioMinutos int `json:"inicioMinutos" validate:"min=0,max=1440"`
	FimMinutos    int `json:"fimMinutos"    validate:"min=0,max=1440"`
}

// DefinirDiaRequest define uma data específica: "bloqueio" deixa o dia
// indisponível (sem blocos); "extra" substitui o expediente padrão pelos
// blocos informados.
type DefinirDiaRequest struct {
	Tipo   string     `json:"tipo" validate:"required,oneof=bloqueio extra"`
	Blocos []BlocoDTO `json:"blocos" validate:"dive"`
}

func (r DefinirDiaRequest) Validar() error {
	return validate.Struct(r)
}

// DiaAgendaDTO é a disponibilidade resolvida de uma data: origem "padrao",
// "bloqueio" ou "extra", com os blocos efetivos do dia.
type DiaAgendaDTO struct {
	Data   string     `json:"data"`
	Origem string     `json:"origem"`
	Blocos []BlocoDTO `json:"blocos"`
}

type AgendaResponse struct {
	AceitaAgendamentos bool           `json:"aceitaAgendamentos"`
	Dias               []DiaAgendaDTO `json:"dias"`
}
