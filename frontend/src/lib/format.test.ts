import { describe, it, expect } from 'vitest';
import { dataLonga, minutosParaHHMM, rotuloStatus } from './format';

describe('minutosParaHHMM', () => {
	it('formata minutos desde a meia-noite com zero à esquerda', () => {
		expect(minutosParaHHMM(8 * 60)).toBe('08:00');
		expect(minutosParaHHMM(9 * 60 + 30)).toBe('09:30');
		expect(minutosParaHHMM(14 * 60 + 5)).toBe('14:05');
	});
});

describe('dataLonga', () => {
	it('formata a data com a primeira letra maiúscula', () => {
		// 2026-08-10 é uma segunda-feira
		expect(dataLonga('2026-08-10')).toMatch(/^Seg/);
	});
});

describe('rotuloStatus', () => {
	it('mapeia os status conhecidos', () => {
		expect(rotuloStatus('SOLICITADO').texto).toBe('Aguardando confirmação');
		expect(rotuloStatus('CONFIRMADO').cor).toBe('bg-accent-green');
	});

	it('devolve o próprio status para valores desconhecidos', () => {
		expect(rotuloStatus('OUTRO').texto).toBe('OUTRO');
	});
});
