import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
	consultarGradeSemanal,
	definirGradeSemanal,
	listarExcecoes,
	criarExcecao,
	removerExcecao
} from './availability';
import { ApiError } from './client';

function mockFetch(resposta: Response) {
	const fn = vi.fn().mockResolvedValue(resposta);
	vi.stubGlobal('fetch', fn);
	return fn;
}

beforeEach(() => {
	vi.unstubAllGlobals();
});

describe('consultarGradeSemanal', () => {
	it('devolve a grade em caso de sucesso', async () => {
		mockFetch(new Response(JSON.stringify({ dias: [] }), { status: 200 }));
		const resultado = await consultarGradeSemanal();
		expect(resultado).toEqual({ dias: [] });
	});

	it('envia GET com credentials include', async () => {
		const fn = mockFetch(new Response(JSON.stringify({ dias: [] }), { status: 200 }));
		await consultarGradeSemanal();
		expect(fn.mock.calls[0][0]).toContain('/providers/me/disponibilidade');
		expect(fn.mock.calls[0][1]).toMatchObject({ credentials: 'include' });
	});
});

describe('definirGradeSemanal', () => {
	it('envia PUT com o corpo correto', async () => {
		const dias = [{ diaSemana: 1, blocos: [{ inicioMinutos: 480, fimMinutos: 720 }] }];
		const fn = mockFetch(new Response(JSON.stringify({ dias }), { status: 200 }));

		const resultado = await definirGradeSemanal(dias);

		expect(resultado).toEqual({ dias });
		expect(fn.mock.calls[0][0]).toContain('/providers/me/disponibilidade');
		expect(fn.mock.calls[0][1]).toMatchObject({ method: 'PUT', credentials: 'include' });
		expect(JSON.parse(fn.mock.calls[0][1].body)).toEqual({ dias });
	});

	it('lança ApiError em caso de bloco inválido', async () => {
		mockFetch(new Response(JSON.stringify({ erro: 'blocos não podem se sobrepor' }), { status: 400 }));
		await expect(definirGradeSemanal([])).rejects.toBeInstanceOf(ApiError);
	});
});

describe('listarExcecoes', () => {
	it('devolve a lista de exceções', async () => {
		mockFetch(new Response(JSON.stringify({ excecoes: [] }), { status: 200 }));
		const resultado = await listarExcecoes();
		expect(resultado).toEqual({ excecoes: [] });
	});
});

describe('criarExcecao', () => {
	it('envia POST e devolve a exceção criada', async () => {
		const excecao = { id: '1', data: '2026-08-10', tipo: 'bloqueio' as const, blocos: [] };
		const fn = mockFetch(new Response(JSON.stringify(excecao), { status: 201 }));

		const resultado = await criarExcecao({ data: '2026-08-10', tipo: 'bloqueio', blocos: [] });

		expect(resultado).toEqual(excecao);
		expect(fn.mock.calls[0][0]).toContain('/providers/me/excecoes');
		expect(fn.mock.calls[0][1]).toMatchObject({ method: 'POST', credentials: 'include' });
	});

	it('propaga 409 quando já existe exceção para a data', async () => {
		mockFetch(
			new Response(JSON.stringify({ erro: 'já existe uma exceção para esta data' }), { status: 409 })
		);
		await expect(
			criarExcecao({ data: '2026-08-10', tipo: 'bloqueio', blocos: [] })
		).rejects.toMatchObject({ status: 409 });
	});
});

describe('removerExcecao', () => {
	it('envia DELETE para o id informado', async () => {
		const fn = mockFetch(new Response(null, { status: 204 }));
		await removerExcecao('exc-1');

		expect(fn.mock.calls[0][0]).toContain('/providers/me/excecoes/exc-1');
		expect(fn.mock.calls[0][1]).toMatchObject({ method: 'DELETE', credentials: 'include' });
	});

	it('propaga 404 quando a exceção não existe', async () => {
		mockFetch(new Response(JSON.stringify({ erro: 'exceção não encontrada' }), { status: 404 }));
		await expect(removerExcecao('id-fantasma')).rejects.toMatchObject({ status: 404 });
	});
});
