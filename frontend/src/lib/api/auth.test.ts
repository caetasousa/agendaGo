import { describe, it, expect, vi, beforeEach } from 'vitest';
import { login } from './auth';

function mockFetchSequence(...respostas: Response[]) {
	const fn = vi.fn();
	respostas.forEach((r) => fn.mockResolvedValueOnce(r));
	vi.stubGlobal('fetch', fn);
	return fn;
}

beforeEach(() => {
	vi.unstubAllGlobals();
});

describe('login (unificado)', () => {
	it('autentica como prestador e não tenta como cliente', async () => {
		const fn = mockFetchSequence(
			new Response(JSON.stringify({ id: '1', nome: 'Ana', tipo: 'provider' }), { status: 200 })
		);
		const resultado = await login({ email: 'ana@email.com', senha: '12345678' });

		expect(resultado.tipo).toBe('provider');
		expect(fn).toHaveBeenCalledTimes(1);
		expect(fn.mock.calls[0][0]).toContain('/auth/provider/login');
	});

	it('cai para cliente quando o login de prestador retorna 401', async () => {
		const fn = mockFetchSequence(
			new Response(JSON.stringify({ erro: 'credenciais inválidas' }), { status: 401 }),
			new Response(JSON.stringify({ id: '2', nome: 'Maria', tipo: 'client' }), { status: 200 })
		);
		const resultado = await login({ email: 'maria@email.com', senha: '12345678' });

		expect(resultado.tipo).toBe('client');
		expect(fn).toHaveBeenCalledTimes(2);
		expect(fn.mock.calls[1][0]).toContain('/auth/client/login');
	});

	it('propaga o erro quando nem prestador nem cliente autenticam', async () => {
		mockFetchSequence(
			new Response(JSON.stringify({ erro: 'credenciais inválidas' }), { status: 401 }),
			new Response(JSON.stringify({ erro: 'credenciais inválidas' }), { status: 401 })
		);

		await expect(login({ email: 'x@email.com', senha: 'errada' })).rejects.toMatchObject({
			status: 401
		});
	});

	it('não tenta login de cliente quando o erro do prestador não é 401', async () => {
		const fn = mockFetchSequence(
			new Response(JSON.stringify({ erro: 'erro interno' }), { status: 500 })
		);

		await expect(login({ email: 'x@email.com', senha: '12345678' })).rejects.toMatchObject({
			status: 500
		});
		expect(fn).toHaveBeenCalledTimes(1);
	});
});
