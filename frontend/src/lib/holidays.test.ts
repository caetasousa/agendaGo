import { describe, it, expect } from 'vitest';
import { chaveData, pascoa, feriadosNacionais } from './holidays';

describe('chaveData', () => {
	it('formata a data no fuso local com zero à esquerda', () => {
		expect(chaveData(new Date(2026, 0, 5))).toBe('2026-01-05');
		expect(chaveData(new Date(2026, 11, 25))).toBe('2026-12-25');
	});
});

describe('pascoa', () => {
	it('calcula o domingo de Páscoa de anos conhecidos', () => {
		expect(chaveData(pascoa(2025))).toBe('2025-04-20');
		expect(chaveData(pascoa(2026))).toBe('2026-04-05');
		expect(chaveData(pascoa(2027))).toBe('2027-03-28');
	});
});

describe('feriadosNacionais', () => {
	it('inclui os feriados fixos', () => {
		const feriados = feriadosNacionais(2026);
		expect(feriados.get('2026-01-01')).toBe('Confraternização Universal');
		expect(feriados.get('2026-09-07')).toBe('Independência do Brasil');
		expect(feriados.get('2026-11-20')).toBe('Dia da Consciência Negra');
		expect(feriados.get('2026-12-25')).toBe('Natal');
	});

	it('calcula os feriados móveis a partir da Páscoa', () => {
		const feriados = feriadosNacionais(2026);
		expect(feriados.get('2026-02-17')).toBe('Carnaval');
		expect(feriados.get('2026-04-03')).toBe('Sexta-feira Santa');
		expect(feriados.get('2026-06-04')).toBe('Corpus Christi');
	});

	it('não marca dias comuns como feriado', () => {
		const feriados = feriadosNacionais(2026);
		expect(feriados.has('2026-07-10')).toBe(false);
	});
});
