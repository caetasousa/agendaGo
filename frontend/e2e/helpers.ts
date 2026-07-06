// Gera um email único por execução para isolar dados entre specs e runs,
// sem depender de reset de banco entre testes.
export function emailUnico(prefixo: string): string {
	const sufixo = Math.random().toString(36).slice(2, 8);
	return `${prefixo}-${Date.now()}-${sufixo}@email.com`;
}
