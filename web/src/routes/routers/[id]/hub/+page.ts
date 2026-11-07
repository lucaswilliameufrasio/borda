import { apiBaseUrl } from '#lib/config.ts';
import { createClient } from '#lib/routers/client.ts';
import { ApiError, type HubView } from '#lib/routers/model.ts';

export type HubPageState = { type: 'error'; message: string } | { type: 'ready'; id: string; view: HubView };

export async function load({ fetch, params }): Promise<{ state: HubPageState }> {
	const client = createClient(apiBaseUrl(), fetch);
	try {
		const view = await client.hub(params.id);
		return { state: { type: 'ready', id: params.id, view } };
	} catch (error) {
		let message = '';
		if (error instanceof ApiError) {
			message = error.message;
		}
		return { state: { type: 'error', message } };
	}
}
