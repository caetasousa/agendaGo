package dto

type BlocoDTO struct {
	InicioMinutos int `json:"inicioMinutos" validate:"min=0,max=1440"`
	FimMinutos    int `json:"fimMinutos"    validate:"min=0,max=1440"`
}

type DiaGradeDTO struct {
	DiaSemana int        `json:"diaSemana" validate:"min=0,max=6"`
	Blocos    []BlocoDTO `json:"blocos"    validate:"dive"`
}

type DefinirGradeSemanalRequest struct {
	Dias []DiaGradeDTO `json:"dias" validate:"dive"`
}

func (r DefinirGradeSemanalRequest) Validar() error {
	return validate.Struct(r)
}

type GradeSemanalResponse struct {
	Dias []DiaGradeDTO `json:"dias"`
}

type CriarExcecaoRequest struct {
	Data   string     `json:"data" validate:"required,datetime=2006-01-02"`
	Tipo   string     `json:"tipo" validate:"required,oneof=bloqueio extra"`
	Blocos []BlocoDTO `json:"blocos" validate:"dive"`
}

func (r CriarExcecaoRequest) Validar() error {
	return validate.Struct(r)
}

type ExcecaoResponse struct {
	ID     string     `json:"id"`
	Data   string     `json:"data"`
	Tipo   string     `json:"tipo"`
	Blocos []BlocoDTO `json:"blocos"`
}

type ListarExcecoesResponse struct {
	Excecoes []ExcecaoResponse `json:"excecoes"`
}
