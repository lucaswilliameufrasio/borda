import { ApiError, type AssistantReply, type HubView, type PathEvent, type ProofResult, type RouterSummary, type SetupDraft, type Snapshot, type WanResult } from './model';
import { parseApiError, parseAssistant, parseEvents, parseHub, parseProof, parseRouter, parseRouterList, parseSnapshot, parseWan } from './parse';
import { buildProofBody, buildRegisterBody, buildUnlockBody, buildWanBody } from './request';

type FetchLike = (input: string, init?: RequestInit) => Promise<Response>;

export function createClient(baseUrl: string, fetchFn: FetchLike) {
	const root = baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl;
	return {
		async listRouters(): Promise<RouterSummary[]> {
			const payload = await requestJson(fetchFn, 'GET', `${root}/v1/routers`);
			return parseRouterList(payload);
		},
		async getRouter(id: string): Promise<RouterSummary> {
			const payload = await requestJson(fetchFn, 'GET', `${root}/v1/routers/${encodeURIComponent(id)}`);
			return parseRouter(payload);
		},
		async createRouter(draft: SetupDraft): Promise<RouterSummary> {
			const payload = await requestJson(fetchFn, 'POST', `${root}/v1/routers`, buildRegisterBody(draft));
			return parseRouter(payload);
		},
		async snapshot(id: string): Promise<Snapshot> {
			const payload = await requestJson(fetchFn, 'GET', `${root}/v1/routers/${encodeURIComponent(id)}/snapshot`);
			return parseSnapshot(payload);
		},
		async events(id: string): Promise<PathEvent[]> {
			const payload = await requestJson(fetchFn, 'GET', `${root}/v1/routers/${encodeURIComponent(id)}/events`);
			return parseEvents(payload);
		},
		async unlock(id: string, passphrase: string, token: string): Promise<void> {
			await requestJson(fetchFn, 'POST', `${root}/v1/routers/${encodeURIComponent(id)}/session`, buildUnlockBody(passphrase, token));
		},
		async prove(id: string): Promise<ProofResult> {
			const payload = await requestJson(fetchFn, 'POST', `${root}/v1/routers/${encodeURIComponent(id)}/proof`, buildProofBody());
			return parseProof(payload);
		},
		async registerWan(id: string, name: string, interfaceName: string, distance: number): Promise<WanResult> {
			const payload = await requestJson(
				fetchFn,
				'POST',
				`${root}/v1/routers/${encodeURIComponent(id)}/wans`,
				buildWanBody(name, interfaceName, distance),
			);
			return parseWan(payload);
		},
		async ask(id: string, question: string): Promise<AssistantReply> {
			const payload = await requestJson(fetchFn, 'POST', `${root}/v1/routers/${encodeURIComponent(id)}/assistant`, { question });
			return parseAssistant(payload);
		},
		async hub(id: string): Promise<HubView> {
			const payload = await requestJson(fetchFn, 'GET', `${root}/v1/routers/${encodeURIComponent(id)}/hub`);
			return parseHub(payload);
		},
		async approveAgent(id: string): Promise<HubView> {
			const payload = await requestJson(fetchFn, 'POST', `${root}/v1/routers/${encodeURIComponent(id)}/link`, { confirmed: true });
			return parseHub(payload);
		},
		async applyProposal(id: string): Promise<ProofResult> {
			const payload = await requestJson(fetchFn, 'POST', `${root}/v1/routers/${encodeURIComponent(id)}/assistant/actions`, {
				action: 'proof',
				confirmed: true,
			});
			return parseProof(payload);
		},
	};
}

async function requestJson(
	fetchFn: FetchLike,
	method: string,
	url: string,
	body?: Record<string, string | number | boolean>,
): Promise<unknown> {
	const headers: Record<string, string> = { accept: 'application/json' };
	let encoded: string | undefined;
	if (body !== undefined) {
		headers['content-type'] = 'application/json';
		encoded = JSON.stringify(body);
	}
	let response: Response;
	try {
		response = await fetchFn(url, { method, headers, body: encoded });
	} catch {
		throw new ApiError('network', 'NETWORK', []);
	}
	if (response.status === 204) {
		return undefined;
	}
	const text = await response.text();
	let payload: unknown;
	if (text === '') {
		payload = undefined;
	} else {
		try {
			payload = JSON.parse(text);
		} catch {
			throw new ApiError('malformed', 'MALFORMED_REQUEST', []);
		}
	}
	if (!response.ok) {
		const parsed = parseApiError(payload);
		if (parsed === undefined) {
			throw new ApiError('request failed', 'UNEXPECTED_ERROR', []);
		}
		throw new ApiError(parsed.message, parsed.errorCode, parsed.fieldErrors);
	}
	return payload;
}
