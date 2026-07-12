// Formatadores compartilhados de horário/data para as telas de agendamento.

// minutosParaHHMM converte minutos desde a meia-noite em "HH:MM".
export function minutosParaHHMM(minutos: number): string {
	const h = Math.floor(minutos / 60)
		.toString()
		.padStart(2, '0');
	const m = (minutos % 60).toString().padStart(2, '0');
	return `${h}:${m}`;
}

// dataLonga formata uma data "YYYY-MM-DD" como "Seg, 10 ago" (pt-BR).
export function dataLonga(data: string): string {
	const [ano, mes, dia] = data.split('-').map(Number);
	const rotulo = new Intl.DateTimeFormat('pt-BR', {
		weekday: 'short',
		day: 'numeric',
		month: 'short'
	}).format(new Date(ano, mes - 1, dia));
	return rotulo.charAt(0).toUpperCase() + rotulo.slice(1);
}

// rotuloStatus devolve o texto e a cor do marcador para cada status do
// agendamento — usado nas listas do admin.
export function rotuloStatus(status: string): { texto: string; cor: string } {
	const mapa: Record<string, { texto: string; cor: string }> = {
		SOLICITADO: { texto: 'Aguardando confirmação', cor: 'bg-accent-yellow' },
		CONFIRMADO: { texto: 'Confirmado', cor: 'bg-accent-green' },
		REALIZADO: { texto: 'Realizado', cor: 'bg-accent-blue' },
		RECUSADO: { texto: 'Recusado', cor: 'bg-accent-red' },
		EXPIRADO: { texto: 'Expirado', cor: 'bg-accent-red' },
		CANCELADO: { texto: 'Cancelado', cor: 'bg-accent-red' },
		NAO_COMPARECEU: { texto: 'Não compareceu', cor: 'bg-accent-orange' }
	};
	return mapa[status] ?? { texto: status, cor: 'bg-mute' };
}
