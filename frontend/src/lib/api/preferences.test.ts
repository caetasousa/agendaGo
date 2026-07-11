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

const horariosPadrao = [{ inicioMinutos: 480, fimMinutos: 720 }];

describe('atualizarPreferencias', () => {
	it('devolve o corpo JSON em caso de sucesso', async () => {
		mockFetch(
			new Response(
				JSON.stringify({ aceitaAgendamentos: true, descansoMinutos: 15, horariosPadrao }),
				{ status: 200 }
			)
		);
		const resultado = await atualizarPreferencias({
			aceitaAgendamentos: true,
			descansoMinutos: 15,
			horariosPadrao
		});
		expect(resultado).toEqual({ aceitaAgendamentos: true, descansoMinutos: 15, horariosPadrao });
	});

	it('envia o método PUT com credentials include e os horários padrão', async () => {
		const fn = mockFetch(
			new Response(
				JSON.stringify({ aceitaAgendamentos: false, descansoMinutos: 0, horariosPadrao: [] }),
				{ status: 200 }
			)
		);
		await atualizarPreferencias({ aceitaAgendamentos: false, descansoMinutos: 0, horariosPadrao: [] });

		expect(fn.mock.calls[0][0]).toContain('/providers/me/preferencias');
		expect(fn.mock.calls[0][1]).toMatchObject({
			method: 'PUT',
			credentials: 'include',
			body: JSON.stringify({ aceitaAgendamentos: false, descansoMinutos: 0, horariosPadrao: [] })
		});
	});

	it('envia múltiplos blocos curtos no corpo da requisição', async () => {
		const tresBlocos = [
			{ inicioMinutos: 480, fimMinutos: 600 },
			{ inicioMinutos: 660, fimMinutos: 780 },
			{ inicioMinutos: 900, fimMinutos: 1020 }
		];
		const fn = mockFetch(
			new Response(
				JSON.stringify({ aceitaAgendamentos: true, descansoMinutos: 10, horariosPadrao: tresBlocos }),
				{ status: 200 }
			)
		);
		await atualizarPreferencias({ aceitaAgendamentos: true, descansoMinutos: 10, horariosPadrao: tresBlocos });

		const corpoEnviado = JSON.parse(fn.mock.calls[0][1].body);
		expect(corpoEnviado.horariosPadrao).toHaveLength(3);
	});

	it('lança ApiError com a mensagem do campo erro em caso de falha', async () => {
		mockFetch(new Response(JSON.stringify({ erro: 'descanso não pode ser negativo' }), { status: 400 }));
		await expect(
			atualizarPreferencias({ aceitaAgendamentos: true, descansoMinutos: -1, horariosPadrao: [] })
		).rejects.toBeInstanceOf(ApiError);
	});

	it('propaga o status 403 quando o usuário não é prestador', async () => {
		mockFetch(
			new Response(JSON.stringify({ erro: 'acesso não permitido para este tipo de usuário' }), {
				status: 403
			})
		);
		await expect(
			atualizarPreferencias({ aceitaAgendamentos: true, descansoMinutos: 0, horariosPadrao: [] })
		).rejects.toMatchObject({ status: 403 });
	});
});
