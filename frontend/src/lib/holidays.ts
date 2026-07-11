// Feriados nacionais brasileiros calculados no cliente: datas fixas mais as
// móveis derivadas da Páscoa (Carnaval, Sexta-feira Santa, Corpus Christi).
// Uso apenas visual no calendário — feriado não bloqueia disponibilidade sozinho.

// chaveData formata uma data como YYYY-MM-DD no fuso local (sem passar por
// toISOString, que deslocaria o dia em fusos negativos como o do Brasil).
export function chaveData(data: Date): string {
	const ano = data.getFullYear();
	const mes = (data.getMonth() + 1).toString().padStart(2, '0');
	const dia = data.getDate().toString().padStart(2, '0');
	return `${ano}-${mes}-${dia}`;
}

// pascoa calcula o domingo de Páscoa de um ano pelo algoritmo de
// Meeus/Jones/Butcher (calendário gregoriano).
export function pascoa(ano: number): Date {
	const a = ano % 19;
	const b = Math.floor(ano / 100);
	const c = ano % 100;
	const d = Math.floor(b / 4);
	const e = b % 4;
	const f = Math.floor((b + 8) / 25);
	const g = Math.floor((b - f + 1) / 3);
	const h = (19 * a + b - d - g + 15) % 30;
	const i = Math.floor(c / 4);
	const k = c % 4;
	const l = (32 + 2 * e + 2 * i - h - k) % 7;
	const m = Math.floor((a + 11 * h + 22 * l) / 451);
	const mes = Math.floor((h + l - 7 * m + 114) / 31);
	const dia = ((h + l - 7 * m + 114) % 31) + 1;
	return new Date(ano, mes - 1, dia);
}

// feriadosNacionais devolve um mapa YYYY-MM-DD → nome com os feriados
// nacionais do ano informado.
export function feriadosNacionais(ano: number): Map<string, string> {
	const domingoPascoa = pascoa(ano);
	const relativoAPascoa = (dias: number) =>
		new Date(ano, domingoPascoa.getMonth(), domingoPascoa.getDate() + dias);

	const feriados = new Map<string, string>();
	feriados.set(`${ano}-01-01`, 'Confraternização Universal');
	feriados.set(chaveData(relativoAPascoa(-47)), 'Carnaval');
	feriados.set(chaveData(relativoAPascoa(-2)), 'Sexta-feira Santa');
	feriados.set(`${ano}-04-21`, 'Tiradentes');
	feriados.set(`${ano}-05-01`, 'Dia do Trabalho');
	feriados.set(chaveData(relativoAPascoa(60)), 'Corpus Christi');
	feriados.set(`${ano}-09-07`, 'Independência do Brasil');
	feriados.set(`${ano}-10-12`, 'Nossa Senhora Aparecida');
	feriados.set(`${ano}-11-02`, 'Finados');
	feriados.set(`${ano}-11-15`, 'Proclamação da República');
	feriados.set(`${ano}-11-20`, 'Dia da Consciência Negra');
	feriados.set(`${ano}-12-25`, 'Natal');
	return feriados;
}
