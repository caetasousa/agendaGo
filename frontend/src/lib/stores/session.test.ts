import { describe, it, expect, vi, beforeEach } from 'vitest';

const meMock = vi.fn();
const logoutMock = vi.fn();

vi.mock('$lib/api/auth', () => ({
	me: (...args: unknown[]) => meMock(...args),
	logout: (...args: unknown[]) => logoutMock(...args)
}));

beforeEach(() => {
	meMock.mockReset();
	logoutMock.mockReset();
});

describe('Sessao', () => {
	it('carregar popula usuario e zera carregando em caso de sucesso', async () => {
		const { Sessao } = await import('./session.svelte');
		meMock.mockResolvedValue({ id: '1', nome: 'Ana', email: 'ana@email.com', tipo: 'provider' });

		const sessao = new Sessao();
		expect(sessao.carregando).toBe(true);

		await sessao.carregar();

		expect(sessao.usuario).toEqual({ id: '1', nome: 'Ana', email: 'ana@email.com', tipo: 'provider' });
		expect(sessao.carregando).toBe(false);
	});

	it('carregar trata falha (401) como deslogado sem lançar', async () => {
		const { Sessao } = await import('./session.svelte');
		meMock.mockRejectedValue(new Error('401'));

		const sessao = new Sessao();
		await expect(sessao.carregar()).resolves.toBeUndefined();

		expect(sessao.usuario).toBeNull();
		expect(sessao.carregando).toBe(false);
	});

	it('carregar não chama me() de novo se usuario já está definido', async () => {
		const { Sessao } = await import('./session.svelte');
		const sessao = new Sessao();
		sessao.definir({ id: '1', nome: 'Ana', email: 'ana@email.com', tipo: 'provider' });

		await sessao.carregar();

		expect(meMock).not.toHaveBeenCalled();
	});

	it('definir popula usuario e zera carregando', async () => {
		const { Sessao } = await import('./session.svelte');
		const sessao = new Sessao();

		sessao.definir({ id: '2', nome: 'Maria', email: 'maria@email.com', tipo: 'client' });

		expect(sessao.usuario?.nome).toBe('Maria');
		expect(sessao.carregando).toBe(false);
	});

	it('sair chama logout e limpa o estado', async () => {
		const { Sessao } = await import('./session.svelte');
		logoutMock.mockResolvedValue(undefined);

		const sessao = new Sessao();
		sessao.definir({ id: '1', nome: 'Ana', email: 'ana@email.com', tipo: 'provider' });

		await sessao.sair();

		expect(logoutMock).toHaveBeenCalledTimes(1);
		expect(sessao.usuario).toBeNull();
	});

	it('sair limpa o estado mesmo se logout falhar', async () => {
		const { Sessao } = await import('./session.svelte');
		logoutMock.mockRejectedValue(new Error('rede fora do ar'));

		const sessao = new Sessao();
		sessao.definir({ id: '1', nome: 'Ana', email: 'ana@email.com', tipo: 'provider' });

		await expect(sessao.sair()).rejects.toThrow('rede fora do ar');
		expect(sessao.usuario).toBeNull();
	});
});
