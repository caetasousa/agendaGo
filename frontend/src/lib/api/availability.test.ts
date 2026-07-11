import { describe, it, expect, vi, beforeEach } from 'vitest';
import { consultarAgenda, definirDia, removerDia } from './availability';
import { ApiError } from './client';

function mockFetch(resposta: Response) {
	const fn = vi.fn().mockResolvedValue(resposta);
	vi.stubGlobal('fetch', fn);
	return fn;
}

beforeEach(() => {
	vi.unstubAllGlobals();
});

describe('consultarAgenda', () => {
	it('devolve a agenda resolvida em caso de sucesso', async () => {
		const agenda = {
			aceitaAgendamentos: true,
			dias: [{ data: '2026-07-10', origem: 'padrao', blocos: [] }]
		};
		mockFetch(new Response(JSON.stringify(agenda), { status: 200 }));
		const resultado = await consultarAgenda('2026-07-01', '2026-07-31');
		expect(resultado).toEqual(agenda);
	});

	it('envia GET com o período e credentials include', async () => {
		const fn = mockFetch(
			new Response(JSON.stringify({ aceitaAgendamentos: true, dias: [] }), { status: 200 })
		);
		await consultarAgenda('2026-07-01', '2026-07-31');
		expect(fn.mock.calls[0][0]).toContain('/providers/me/agenda?de=2026-07-01&ate=2026-07-31');
		expect(fn.mock.calls[0][1]).toMatchObject({ credentials: 'include' });
	});

	it('lança ApiError com a mensagem da API em caso de erro', async () => {
		mockFetch(new Response(JSON.stringify({ erro: 'período inválido' }), { status: 400 }));
		await expect(consultarAgenda('2026-07-31', '2026-07-01')).rejects.toThrowError(
			new ApiError(400, 'período inválido')
		);
	});
});

describe('definirDia', () => {
	it('envia PUT para a data com o corpo correto', async () => {
		const dia = { data: '2026-07-15', origem: 'bloqueio', blocos: [] };
		const fn = mockFetch(new Response(JSON.stringify(dia), { status: 200 }));

		const resultado = await definirDia('2026-07-15', { tipo: 'bloqueio', blocos: [] });
		expect(resultado).toEqual(dia);
		expect(fn.mock.calls[0][0]).toContain('/providers/me/dias/2026-07-15');
		expect(fn.mock.calls[0][1]).toMatchObject({
			method: 'PUT',
			body: JSON.stringify({ tipo: 'bloqueio', blocos: [] })
		});
	});

	it('envia blocos quando o tipo é extra', async () => {
		const blocos = [{ inicioMinutos: 480, fimMinutos: 720 }];
		const fn = mockFetch(
			new Response(JSON.stringify({ data: '2026-07-18', origem: 'extra', blocos }), { status: 200 })
		);

		await definirDia('2026-07-18', { tipo: 'extra', blocos });
		expect(fn.mock.calls[0][1]).toMatchObject({ body: JSON.stringify({ tipo: 'extra', blocos }) });
	});
});

describe('removerDia', () => {
	it('envia DELETE para a data', async () => {
		const fn = mockFetch(new Response(null, { status: 204 }));
		await removerDia('2026-07-15');
		expect(fn.mock.calls[0][0]).toContain('/providers/me/dias/2026-07-15');
		expect(fn.mock.calls[0][1]).toMatchObject({ method: 'DELETE' });
	});

	it('lança ApiError quando a data não tem definição própria', async () => {
		mockFetch(new Response(JSON.stringify({ erro: 'não há definição própria para esta data' }), { status: 404 }));
		await expect(removerDia('2030-01-01')).rejects.toThrowError(
			new ApiError(404, 'não há definição própria para esta data')
		);
	});
});
