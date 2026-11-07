import { apiBaseUrl } from '#lib/config.ts';
import { createClient } from '#lib/routers/client.ts';
import { ApiError, type RouterSummary } from '#lib/routers/model.ts';

export type ListState =
	| { type: 'ready'; routers: RouterSummary[] }
	| { type: 'error'; message: string };

export async function load({ fetch }): Promise<{ page: ListState }> {
	try {
		const routers = await createClient(apiBaseUrl(), fetch).listRouters();
		return { page: { type: 'ready', routers } };
	} catch (error) {
		if (error instanceof ApiError && error.message !== '' && error.errorCode !== 'NETWORK') {
			return { page: { type: 'error', message: error.message } };
		}
		return { page: { type: 'error', message: '' } };
	}
}
