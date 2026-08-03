import { describe, it, expect, vi, beforeEach } from 'vitest';
import { buscarMetricas } from './metricas';
import { ApiError } from './client';

function mockFetch(resposta: Response) {
	const fn = vi.fn().mockResolvedValue(resposta);
	vi.stubGlobal('fetch', fn);
	return fn;
}

beforeEach(() => {
	vi.unstubAllGlobals();
});

const resumo = {
	de: '2026-07-04',
	ate: '2026-08-02',
	porStatus: {
		SOLICITADO: 2,
		CONFIRMADO: 5,
		REALIZADO: 21,
		NAO_COMPARECEU: 3,
		CANCELADO: 1,
		RECUSADO: 0,
		EXPIRADO: 0
	},
	total: 32,
	minutosOfertados: 9600,
	minutosReservados: 1740,
	taxaOcupacao: 0.18125,
	taxaComparecimento: 0.875
};

describe('buscarMetricas', () => {
	it('envia GET com o período e devolve o resumo', async () => {
		const fn = mockFetch(new Response(JSON.stringify(resumo), { status: 200 }));

		const resultado = await buscarMetricas('2026-07-04', '2026-08-02');

		expect(fn.mock.calls[0][0]).toContain('/providers/me/metricas?de=2026-07-04&ate=2026-08-02');
		expect(fn.mock.calls[0][1]).toMatchObject({ credentials: 'include' });
		expect(resultado.total).toBe(32);
		expect(resultado.porStatus.REALIZADO).toBe(21);
	});

	it('preserva taxa nula: sem base para medir não é zero', async () => {
		const semBase = { ...resumo, taxaOcupacao: null, taxaComparecimento: null };
		mockFetch(new Response(JSON.stringify(semBase), { status: 200 }));

		const resultado = await buscarMetricas('2026-07-04', '2026-08-02');

		expect(resultado.taxaOcupacao).toBeNull();
		expect(resultado.taxaComparecimento).toBeNull();
	});

	it('propaga erro da API', async () => {
		mockFetch(new Response(JSON.stringify({ erro: 'período inválido' }), { status: 400 }));

		await expect(buscarMetricas('2026-08-02', '2026-07-04')).rejects.toThrow(ApiError);
	});
});
