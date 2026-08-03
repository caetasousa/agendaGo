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

// porcentagem formata uma fração em [0,1] como "62%". Recebe null quando a API
// não teve base para calcular a taxa e devolve um travessão: exibir "0%" ali
// afirmaria uma medida que ninguém fez.
export function porcentagem(fracao: number | null): string {
	if (fracao === null) {
		return '—';
	}
	return `${Math.round(fracao * 100)}%`;
}

// duracaoEmHoras formata minutos como "18h", "18h30" ou "45min" — a leitura de
// quem fala de expediente, não de um cronômetro.
export function duracaoEmHoras(minutos: number): string {
	const horas = Math.floor(minutos / 60);
	const resto = minutos % 60;
	if (horas === 0) {
		return `${resto}min`;
	}
	return resto === 0 ? `${horas}h` : `${horas}h${resto.toString().padStart(2, '0')}`;
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
