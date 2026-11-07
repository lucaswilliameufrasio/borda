import { apiBaseUrl } from '#lib/config.ts';
import { createClient } from '#lib/routers/client.ts';
import { findDevice } from '#lib/routers/parse.ts';
import { ApiError, type ActiveRoute, type Device, type RouterSummary } from '#lib/routers/model.ts';

export type DeviceState =
	| { type: 'error'; message: string }
	| { type: 'missing'; router: RouterSummary }
	| {
			type: 'session-failed';
			router: RouterSummary;
			session: 'unreachable' | 'authentication_rejected' | 'locked';
	  }
	| { type: 'ready'; router: RouterSummary; device: Device; activeRoute?: ActiveRoute };

const macPattern = /^[0-9a-f]{12}$/;

export async function load({ fetch, params }): Promise<{ state: DeviceState }> {
	const client = createClient(apiBaseUrl(), fetch);
	try {
		const router = await client.getRouter(params.id);
		if (!macPattern.test(params.mac)) {
			return { state: { type: 'missing', router } };
		}
		const snapshot = await client.snapshot(params.id);
		if (
			snapshot.session === 'unreachable' ||
			snapshot.session === 'authentication_rejected' ||
			snapshot.session === 'locked'
		) {
			return { state: { type: 'session-failed', router, session: snapshot.session } };
		}
		const device = findDevice(snapshot.devices, params.mac);
		if (device === undefined) {
			return { state: { type: 'missing', router } };
		}
		const ready: DeviceState = { type: 'ready', router, device };
		if (snapshot.activeRoute !== undefined) {
			ready.activeRoute = snapshot.activeRoute;
		}
		return { state: ready };
	} catch (error) {
		let message = '';
		if (error instanceof ApiError) {
			message = error.message;
		}
		return { state: { type: 'error', message } };
	}
}
