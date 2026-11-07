import { apiBaseUrl } from '#lib/config.ts';
import { createClient } from '#lib/routers/client.ts';
import { detailView } from '#lib/routers/parse.ts';
import { ApiError, type DetailView, type RouterSummary } from '#lib/routers/model.ts';

export type DetailState =
	| { type: 'ready'; router: RouterSummary; view: DetailView }
	| { type: 'error'; message: string };

export async function load({ fetch, params }): Promise<{ state: DetailState }> {
	const client = createClient(apiBaseUrl(), fetch);
	try {
		const router = await client.getRouter(params.id);
		const snapshot = await client.snapshot(params.id);
		const state: DetailState = { type: 'ready', router, view: detailView(snapshot) };
		return { state };
	} catch (error) {
		let message = '';
		if (error instanceof ApiError) {
			message = error.message;
		}
		const state: DetailState = { type: 'error', message };
		return { state };
	}
}
