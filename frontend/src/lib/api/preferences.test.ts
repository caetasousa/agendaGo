import { describe, it, expect, vi, beforeEach } from 'vitest';
import { atualizarPreferencias } from './preferences';
import { ApiError } from './client';

function mockFetch(resposta: Response) {
	const fn = vi.fn().mockResolvedValue(resposta);
	vi.stubGlobal('fetch', fn);
	return fn;
}

beforeEach(() => {
	vi.unstubAllGlobals();
});

describe('atualizarPreferencias', () => {
	it('devolve o corpo JSON em caso de sucesso', async () => {
		mockFetch(
			new Response(JSON.stringify({ aceitaAgendamentos: true, descansoMinutos: 15 }), {
				status: 200
			})
		);
		const resultado = await atualizarPreferencias({
			aceitaAgendamentos: true,
			descansoMinutos: 15
		});
		expect(resultado).toEqual({ aceitaAgendamentos: true, descansoMinutos: 15 });
	});

	it('envia o método PUT com credentials include', async () => {
		const fn = mockFetch(
			new Response(JSON.stringify({ aceitaAgendamentos: false, descansoMinutos: 0 }), {
				status: 200
			})
		);
		await atualizarPreferencias({ aceitaAgendamentos: false, descansoMinutos: 0 });

		expect(fn.mock.calls[0][0]).toContain('/providers/me/preferencias');
		expect(fn.mock.calls[0][1]).toMatchObject({ method: 'PUT', credentials: 'include' });
	});

	it('lança ApiError com a mensagem do campo erro em caso de falha', async () => {
		mockFetch(new Response(JSON.stringify({ erro: 'descanso não pode ser negativo' }), { status: 400 }));
		await expect(
			atualizarPreferencias({ aceitaAgendamentos: true, descansoMinutos: -1 })
		).rejects.toBeInstanceOf(ApiError);
	});

	it('propaga o status 403 quando o usuário não é prestador', async () => {
		mockFetch(
			new Response(JSON.stringify({ erro: 'acesso não permitido para este tipo de usuário' }), {
				status: 403
			})
		);
		await expect(
			atualizarPreferencias({ aceitaAgendamentos: true, descansoMinutos: 0 })
		).rejects.toMatchObject({ status: 403 });
	});
});
