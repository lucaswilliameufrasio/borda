import { apiBaseUrl } from '#lib/config.ts';
import { createClient } from '#lib/routers/client.ts';
import { ApiError, type RouterSummary } from '#lib/routers/model.ts';

export type ProofLoad =
	| { type: 'ready'; router: RouterSummary }
	| { type: 'error'; message: string };

export async function load({ fetch, params }): Promise<{ state: ProofLoad }> {
	const client = createClient(apiBaseUrl(), fetch);
	try {
		const router = await client.getRouter(params.id);
		const state: ProofLoad = { type: 'ready', router };
		return { state };
	} catch (error) {
		let message = '';
		if (error instanceof ApiError) {
			message = error.message;
		}
		const state: ProofLoad = { type: 'error', message };
		return { state };
	}
}
