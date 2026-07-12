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

	it('cai para admin quando prestador e cliente retornam 401', async () => {
		const fn = mockFetchSequence(
			new Response(JSON.stringify({ erro: 'credenciais inválidas' }), { status: 401 }),
			new Response(JSON.stringify({ erro: 'credenciais inválidas' }), { status: 401 }),
			new Response(JSON.stringify({ id: '3', nome: 'Admin', tipo: 'admin' }), { status: 200 })
		);
		const resultado = await login({ email: 'admin@agendago.dev', senha: '12345678' });

		expect(resultado.tipo).toBe('admin');
		expect(fn).toHaveBeenCalledTimes(3);
		expect(fn.mock.calls[2][0]).toContain('/auth/admin/login');
	});

	it('propaga o 401 quando nenhum dos três tipos autentica', async () => {
		mockFetchSequence(
			new Response(JSON.stringify({ erro: 'credenciais inválidas' }), { status: 401 }),
			new Response(JSON.stringify({ erro: 'credenciais inválidas' }), { status: 401 }),
			new Response(JSON.stringify({ erro: 'credenciais inválidas' }), { status: 401 })
		);

		await expect(login({ email: 'x@email.com', senha: 'errada' })).rejects.toMatchObject({
			status: 401
		});
	});

	it('para no 403 (usuário banido) sem tentar os próximos tipos', async () => {
		const fn = mockFetchSequence(
			new Response(JSON.stringify({ erro: 'usuário desativado' }), { status: 403 })
		);

		await expect(login({ email: 'banido@email.com', senha: '12345678' })).rejects.toMatchObject({
			status: 403
		});
		expect(fn).toHaveBeenCalledTimes(1);
	});

	it('não tenta os próximos quando o erro do prestador não é 401', async () => {
		const fn = mockFetchSequence(
			new Response(JSON.stringify({ erro: 'erro interno' }), { status: 500 })
		);

		await expect(login({ email: 'x@email.com', senha: '12345678' })).rejects.toMatchObject({
			status: 500
		});
		expect(fn).toHaveBeenCalledTimes(1);
	});
});
