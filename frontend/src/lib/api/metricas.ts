// Tipos e chamadas da API de métricas do prestador.
// Espelham backend/internal/adapter/http/dto/analytics.go

import { apiGet } from './client';
import type { StatusAgendamento } from './appointments';

export interface Metricas {
	de: string;
	ate: string;
	// Uma entrada por status do ciclo de vida, inclusive as zeradas — a API
	// manda o funil completo para a tela não precisar conhecer a lista.
	porStatus: Record<StatusAgendamento, number>;
	total: number;
	minutosOfertados: number;
	minutosReservados: number;
	// Frações em [0,1]. null quer dizer "não havia o que medir" (expediente
	// zerado, nenhum atendimento concluído) — diferente de 0, que é uma medida.
	taxaOcupacao: number | null;
	taxaComparecimento: number | null;
}

// buscarMetricas resume o período (datas YYYY-MM-DD, inclusivas) da agenda do
// prestador autenticado: funil de status, ocupação e comparecimento.
export function buscarMetricas(de: string, ate: string): Promise<Metricas> {
	return apiGet<Metricas>(`/providers/me/metricas?de=${de}&ate=${ate}`);
}
