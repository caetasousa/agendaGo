import { describe, it, expect, vi, beforeEach } from 'vitest';
import { apiGet, apiPost, apiPostVazio, ApiError } from './client';

function mockFetch(resposta: Response) {
	const fn = vi.fn().mockResolvedValue(resposta);
	vi.stubGlobal('fetch', fn);
	return fn;
}

beforeEach(() => {
	vi.unstubAllGlobals();
});

describe('apiPost', () => {
	it('devolve o corpo JSON em caso de sucesso', async () => {
		mockFetch(new Response(JSON.stringify({ id: '1' }), { status: 201 }));
		const resultado = await apiPost('/providers', { nome: 'Ana' });
		expect(resultado).toEqual({ id: '1' });
	});

	it('lança ApiError com a mensagem do campo erro em caso de falha', async () => {
		mockFetch(new Response(JSON.stringify({ erro: 'email já cadastrado' }), { status: 409 }));
		await expect(apiPost('/providers', {})).rejects.toMatchObject({
			status: 409,
			message: 'email já cadastrado'
		});
	});

	it('lança ApiError com mensagem padrão quando o corpo não é JSON', async () => {
		mockFetch(new Response('não é json', { status: 500 }));
		await expect(apiPost('/providers', {})).rejects.toMatchObject({
			status: 500,
			message: 'erro 500'
		});
	});

	it('envia credentials include', async () => {
		const fn = mockFetch(new Response(JSON.stringify({}), { status: 200 }));
		await apiPost('/providers', {});
		expect(fn.mock.calls[0][1]).toMatchObject({ credentials: 'include' });
	});
});

describe('apiGet', () => {
	it('devolve o corpo JSON em caso de sucesso', async () => {
		mockFetch(new Response(JSON.stringify({ nome: 'Ana' }), { status: 200 }));
		const resultado = await apiGet('/auth/me');
		expect(resultado).toEqual({ nome: 'Ana' });
	});

	it('lança ApiError em caso de falha', async () => {
		mockFetch(new Response(JSON.stringify({ erro: 'não autenticado' }), { status: 401 }));
		await expect(apiGet('/auth/me')).rejects.toBeInstanceOf(ApiError);
	});

	it('envia credentials include', async () => {
		const fn = mockFetch(new Response(JSON.stringify({}), { status: 200 }));
		await apiGet('/auth/me');
		expect(fn.mock.calls[0][1]).toMatchObject({ credentials: 'include' });
	});
});

describe('apiPostVazio', () => {
	it('resolve sem corpo em caso de sucesso (204)', async () => {
		mockFetch(new Response(null, { status: 204 }));
		await expect(apiPostVazio('/auth/logout')).resolves.toBeUndefined();
	});

	it('lança ApiError em caso de falha', async () => {
		mockFetch(new Response(JSON.stringify({ erro: 'erro interno' }), { status: 500 }));
		await expect(apiPostVazio('/auth/logout')).rejects.toMatchObject({ status: 500 });
	});

	it('envia credentials include', async () => {
		const fn = mockFetch(new Response(null, { status: 204 }));
		await apiPostVazio('/auth/logout');
		expect(fn.mock.calls[0][1]).toMatchObject({ credentials: 'include' });
	});
});
