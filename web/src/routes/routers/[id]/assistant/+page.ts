import { apiBaseUrl } from '#lib/config.ts';
import { createClient } from '#lib/routers/client.ts';
import { ApiError, type RouterSummary } from '#lib/routers/model.ts';

export type AssistantPageState =
	| { type: 'error'; message: string }
	| { type: 'ready'; router: RouterSummary };

export async function load({ fetch, params }): Promise<{ state: AssistantPageState }> {
	const client = createClient(apiBaseUrl(), fetch);
	try {
		const router = await client.getRouter(params.id);
		return { state: { type: 'ready', router } };
	} catch (error) {
		let message = '';
		if (error instanceof ApiError) {
			message = error.message;
		}
		return { state: { type: 'error', message } };
	}
}
