import { apiBaseUrl } from '#lib/config.ts';
import { createClient } from '#lib/routers/client.ts';
import { ApiError, type PathEvent, type RouterSummary } from '#lib/routers/model.ts';

export type HistoryState =
	| { type: 'error'; message: string }
	| { type: 'ready'; router: RouterSummary; events: PathEvent[] };

export async function load({ fetch, params }): Promise<{ state: HistoryState }> {
	const client = createClient(apiBaseUrl(), fetch);
	try {
		const router = await client.getRouter(params.id);
		const events = await client.events(params.id);
		return { state: { type: 'ready', router, events } };
	} catch (error) {
		let message = '';
		if (error instanceof ApiError) {
			message = error.message;
		}
		return { state: { type: 'error', message } };
	}
}
