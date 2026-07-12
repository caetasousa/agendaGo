// Package slot calcula os horários livres ofertáveis de um dia — cálculo
// puro, sob demanda, nunca pré-gravado:
//
//	slots livres = blocos do dia − intervalos ocupados
//
// fatiados pela duração do atendimento + buffer do prestador, com a sobra
// no fim do bloco descartada (um atendimento nunca vaza para fora do bloco).
package slot

import "agendago/internal/domain/availability"

// Slot é um horário livre ofertável, em minutos desde a meia-noite.
type Slot struct {
	InicioMinutos int
	FimMinutos    int
}

// Intervalo é um trecho ocupado do dia (agendamento SOLICITADO não expirado
// ou CONFIRMADO), em minutos desde a meia-noite.
type Intervalo struct {
	InicioMinutos int
	FimMinutos    int
}

// Livres fatia cada bloco do dia em slots de duracaoMinutos, abrindo o
// próximo slot só após duracao+buffer, e descarta os que colidem com um
// intervalo ocupado (considerando o buffer de preparação após cada ocupação).
// Devolve vazio para duração não positiva ou buffer negativo.
func Livres(blocos []availability.TimeBlock, ocupados []Intervalo, duracaoMinutos, bufferMinutos int) []Slot {
	if duracaoMinutos <= 0 || bufferMinutos < 0 {
		return nil
	}

	passo := duracaoMinutos + bufferMinutos
	var livres []Slot
	for _, bloco := range blocos {
		for inicio := bloco.InicioMinutos; inicio+duracaoMinutos <= bloco.FimMinutos; inicio += passo {
			candidato := Slot{InicioMinutos: inicio, FimMinutos: inicio + duracaoMinutos}
			if !colide(candidato, ocupados, bufferMinutos) {
				livres = append(livres, candidato)
			}
		}
	}
	return livres
}

// colide verifica se o slot cruza algum intervalo ocupado, estendendo cada
// ocupação até fim+buffer — o próximo atendimento só pode começar depois do
// tempo de preparação/limpeza.
func colide(s Slot, ocupados []Intervalo, bufferMinutos int) bool {
	for _, o := range ocupados {
		if s.InicioMinutos < o.FimMinutos+bufferMinutos && o.InicioMinutos < s.FimMinutos+bufferMinutos {
			return true
		}
	}
	return false
}
