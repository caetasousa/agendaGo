// Parâmetros de negócio do agendamento, centralizados como o doc de regras
// exige: fuso único fixo, TTL da pendência e antecedência mínima de
// cancelamento. Os usecases recebem esses valores por injeção — o domínio
// nunca importa config.
package config

import "time"

// TTLSolicitacao é o prazo para o prestador confirmar uma solicitação antes
// de ela expirar e liberar o intervalo.
const TTLSolicitacao = 24 * time.Hour

// AntecedenciaMinimaCancelamento é o prazo mínimo antes do início do
// atendimento em que um agendamento confirmado ainda pode ser cancelado.
const AntecedenciaMinimaCancelamento = 24 * time.Hour

// FusoHorario é o fuso único do sistema (America/Sao_Paulo). Falha no boot se
// o tzdata não estiver disponível — melhor que operar em fuso errado.
var FusoHorario = carregarFuso()

func carregarFuso() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		panic("config: fuso America/Sao_Paulo indisponível: " + err.Error())
	}
	return loc
}
