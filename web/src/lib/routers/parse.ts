import {
	ParseError,
	type ActiveRoute,
	type CheckResult,
	type DetailView,
	type Device,
	type FieldError,
	type ForwardRule,
	type Lease,
	type AssistantKind,
	type AssistantProposal,
	type AssistantReply,
	type PathEvent,
	type PathReason,
	type ProofResult,
	type Reserve,
	type RouteRow,
	type RouterSummary,
	type SessionName,
	type Snapshot,
	type VaultKind,
} from './model';

export function isRecord(value: unknown): value is Record<string, unknown> {
	return Object.prototype.toString.call(value) === '[object Object]';
}

export function parseRouterList(value: unknown): RouterSummary[] {
	if (!Array.isArray(value)) {
		throw new ParseError('routers');
	}
	const routers: RouterSummary[] = [];
	for (const item of value) {
		routers.push(parseRouter(item));
	}
	return routers;
}

export function parseRouter(value: unknown): RouterSummary {
	if (!isRecord(value)) {
		throw new ParseError('router');
	}
	const checkAddress = readOptionalString(value, 'check_address');
	const router: RouterSummary = {
		id: requireString(value, 'id'),
		name: requireString(value, 'name'),
		host: requireString(value, 'host'),
		sshPort: requireNumber(value, 'ssh_port'),
		vaultKind: readVaultKind(requireString(value, 'vault_kind')),
	};
	if (checkAddress !== undefined) {
		router.checkAddress = checkAddress;
	}
	return router;
}

export function parseSnapshot(value: unknown): Snapshot {
	if (!isRecord(value)) {
		throw new ParseError('snapshot');
	}
	const session = readSession(requireString(value, 'session'));
	const snapshot: Snapshot = {
		routerId: requireString(value, 'router_id'),
		name: requireString(value, 'name'),
		session,
		routes: readRoutes(value),
		forwardRules: readForward(value),
		leases: readLeases(value),
		devices: readDevices(value),
	};
	const identity = readOptionalRecord(value, 'identity');
	if (identity !== undefined) {
		const name = readOptionalString(identity, 'name');
		const version = readOptionalString(identity, 'version');
		snapshot.identity = {};
		if (name !== undefined) {
			snapshot.identity.name = name;
		}
		if (version !== undefined) {
			snapshot.identity.version = version;
		}
	}
	const activeRoute = readOptionalRecord(value, 'active_route');
	if (activeRoute !== undefined) {
		snapshot.activeRoute = readActiveRoute(activeRoute);
	}
	const check = readOptionalRecord(value, 'check');
	if (check !== undefined) {
		snapshot.check = readCheck(check);
	}
	const reserve = readOptionalRecord(value, 'reserve');
	if (reserve !== undefined) {
		snapshot.reserve = readReserve(reserve);
	}
	return snapshot;
}

export function parseProof(value: unknown): ProofResult {
	if (!isRecord(value)) {
		throw new ParseError('proof');
	}
	const outcome = requireString(value, 'outcome');
	if (outcome !== 'passed' && outcome !== 'failed') {
		throw new ParseError('outcome');
	}
	return {
		outcome,
		primaryActive: requireBoolean(value, 'primary_active'),
		usbReceived: requireNumber(value, 'usb_received'),
		lanReceived: requireNumber(value, 'lan_received'),
	};
}

export function detailView(snapshot: Snapshot): DetailView {
	if (snapshot.session === 'unreachable' || snapshot.session === 'authentication_rejected' || snapshot.session === 'locked') {
		return { type: 'session-failed', session: snapshot.session };
	}
	if (snapshot.activeRoute === undefined) {
		return { type: 'empty-route', snapshot };
	}
	return { type: 'connected', snapshot };
}

export function fieldMessage(errors: FieldError[], field: string): string | undefined {
	for (const error of errors) {
		if (error.field === field) {
			return error.message;
		}
	}
	return undefined;
}

export function parseApiError(value: unknown): { message: string; errorCode: string; fieldErrors: FieldError[] } | undefined {
	if (!isRecord(value)) {
		return undefined;
	}
	const message = readOptionalString(value, 'message');
	const errorCode = readOptionalString(value, 'error_code');
	if (message === undefined || errorCode === undefined) {
		return undefined;
	}
	return { message, errorCode, fieldErrors: readFieldErrors(value) };
}

function readFieldErrors(value: Record<string, unknown>): FieldError[] {
	const extra = readOptionalRecord(value, 'extra');
	if (extra === undefined) {
		return [];
	}
	const raw = extra.validation_errors;
	if (raw === undefined) {
		return [];
	}
	if (!Array.isArray(raw)) {
		throw new ParseError('validation_errors');
	}
	const errors: FieldError[] = [];
	for (const item of raw) {
		if (!isRecord(item)) {
			throw new ParseError('validation_errors');
		}
		errors.push({
			field: requireString(item, 'field'),
			message: requireString(item, 'message'),
		});
	}
	return errors;
}

function readRoutes(value: Record<string, unknown>): RouteRow[] {
	const rows = readOptionalArray(value, 'routes');
	const routes: RouteRow[] = [];
	for (const item of rows) {
		if (!isRecord(item)) {
			throw new ParseError('routes');
		}
		const route: RouteRow = {
			gateway: requireString(item, 'gateway'),
			active: requireBoolean(item, 'active'),
			distance: requireNumber(item, 'distance'),
		};
		const interfaceName = readOptionalString(item, 'interface_name');
		if (interfaceName !== undefined) {
			route.interfaceName = interfaceName;
		}
		routes.push(route);
	}
	return routes;
}

function readForward(value: Record<string, unknown>): ForwardRule[] {
	const rows = readOptionalArray(value, 'forward_rules');
	const rules: ForwardRule[] = [];
	for (const item of rows) {
		if (!isRecord(item)) {
			throw new ParseError('forward_rules');
		}
		const rule: ForwardRule = { action: requireString(item, 'action') };
		const comment = readOptionalString(item, 'comment');
		const inInterface = readOptionalString(item, 'in_interface');
		const outInterface = readOptionalString(item, 'out_interface');
		if (comment !== undefined) {
			rule.comment = comment;
		}
		if (inInterface !== undefined) {
			rule.inInterface = inInterface;
		}
		if (outInterface !== undefined) {
			rule.outInterface = outInterface;
		}
		rules.push(rule);
	}
	return rules;
}

function readLeases(value: Record<string, unknown>): Lease[] {
	const rows = readOptionalArray(value, 'leases');
	const leases: Lease[] = [];
	for (const item of rows) {
		if (!isRecord(item)) {
			throw new ParseError('leases');
		}
		const lease: Lease = {
			address: requireString(item, 'address'),
			mac: requireString(item, 'mac'),
		};
		const hostName = readOptionalString(item, 'host_name');
		if (hostName !== undefined) {
			lease.hostName = hostName;
		}
		leases.push(lease);
	}
	return leases;
}

function readDevices(value: Record<string, unknown>): Device[] {
	const rows = readOptionalArray(value, 'devices');
	const devices: Device[] = [];
	for (const item of rows) {
		if (!isRecord(item)) {
			throw new ParseError('devices');
		}
		const device: Device = {
			id: requireString(item, 'id'),
			mac: requireString(item, 'mac'),
		};
		const address = readOptionalString(item, 'address');
		const name = readOptionalString(item, 'name');
		const seenOn = readOptionalString(item, 'seen_on');
		if (address !== undefined) {
			device.address = address;
		}
		if (name !== undefined) {
			device.name = name;
		}
		if (seenOn !== undefined) {
			device.seenOn = seenOn;
		}
		devices.push(device);
	}
	return devices;
}

export function parseEvents(value: unknown): PathEvent[] {
	if (!Array.isArray(value)) {
		throw new ParseError('events');
	}
	const events: PathEvent[] = [];
	for (const item of value) {
		if (!isRecord(item)) {
			throw new ParseError('events');
		}
		events.push({
			id: requireString(item, 'id'),
			at: requireString(item, 'at'),
			reason: readReason(requireString(item, 'reason')),
		});
	}
	return events;
}

function readReason(value: string): PathReason {
	if (value === 'check_failed' || value === 'proof_requested' || value === 'route_restored') {
		return value;
	}
	throw new ParseError('reason');
}

export function parseAssistant(value: unknown): AssistantReply {
	if (!isRecord(value)) {
		throw new ParseError('assistant');
	}
	const kind = readAssistantKind(requireString(value, 'kind'));
	const text = requireString(value, 'text');
	const missing = readStringList(value, 'missing');
	if (kind === 'proposal') {
		return { kind, text, missing, proposal: readProposal(value.proposal) };
	}
	if (value.proposal !== undefined) {
		throw new ParseError('proposal');
	}
	return { kind, text, missing };
}

function readAssistantKind(value: string): AssistantKind {
	if (value === 'answer' || value === 'proposal' || value === 'missing' || value === 'refused') {
		return value;
	}
	throw new ParseError('kind');
}

function readProposal(value: unknown): AssistantProposal {
	if (!isRecord(value)) {
		throw new ParseError('proposal');
	}
	const action = requireString(value, 'action');
	if (action !== 'proof') {
		throw new ParseError('action');
	}
	return { action, summary: requireString(value, 'summary') };
}

function readStringList(record: Record<string, unknown>, key: string): string[] {
	const value = record[key];
	if (value === undefined) {
		return [];
	}
	if (!Array.isArray(value)) {
		throw new ParseError(key);
	}
	const items: string[] = [];
	for (const item of value) {
		if (typeof item !== 'string' || item === '') {
			throw new ParseError(key);
		}
		items.push(item);
	}
	return items;
}

export function findDevice(devices: Device[], id: string): Device | undefined {
	for (const device of devices) {
		if (device.id === id) {
			return device;
		}
	}
	return undefined;
}

function readActiveRoute(value: Record<string, unknown>): ActiveRoute {
	const route: ActiveRoute = {
		gateway: requireString(value, 'gateway'),
		distance: requireNumber(value, 'distance'),
	};
	const interfaceName = readOptionalString(value, 'interface_name');
	if (interfaceName !== undefined) {
		route.interfaceName = interfaceName;
	}
	return route;
}

function readCheck(value: Record<string, unknown>): CheckResult {
	return {
		address: requireString(value, 'address'),
		responding: requireBoolean(value, 'responding'),
	};
}

function readReserve(value: Record<string, unknown>): Reserve {
	return {
		interfaceName: requireString(value, 'interface_name'),
		running: requireBoolean(value, 'running'),
		hasAddress: requireBoolean(value, 'has_address'),
	};
}

function readOptionalArray(value: Record<string, unknown>, key: string): unknown[] {
	const raw = value[key];
	if (raw === undefined) {
		return [];
	}
	if (!Array.isArray(raw)) {
		throw new ParseError(key);
	}
	return raw;
}

function readOptionalRecord(value: Record<string, unknown>, key: string): Record<string, unknown> | undefined {
	const raw = value[key];
	if (raw === undefined) {
		return undefined;
	}
	if (!isRecord(raw)) {
		throw new ParseError(key);
	}
	return raw;
}

function readOptionalString(value: Record<string, unknown>, key: string): string | undefined {
	const raw = value[key];
	if (raw === undefined) {
		return undefined;
	}
	if (typeof raw !== 'string') {
		throw new ParseError(key);
	}
	return raw;
}

function requireString(value: Record<string, unknown>, key: string): string {
	const raw = readOptionalString(value, key);
	if (raw === undefined || raw === '') {
		throw new ParseError(key);
	}
	return raw;
}

function requireNumber(value: Record<string, unknown>, key: string): number {
	const raw = value[key];
	if (typeof raw !== 'number' || !Number.isFinite(raw)) {
		throw new ParseError(key);
	}
	return raw;
}

function requireBoolean(value: Record<string, unknown>, key: string): boolean {
	const raw = value[key];
	if (typeof raw !== 'boolean') {
		throw new ParseError(key);
	}
	return raw;
}

function readSession(value: string): SessionName {
	if (value === 'connected' || value === 'unreachable' || value === 'authentication_rejected' || value === 'locked') {
		return value;
	}
	throw new ParseError('session');
}

function readVaultKind(value: string): VaultKind {
	if (value === 'file' || value === 'keychain' || value === 'ciphera' || value === 'infisical') {
		return value;
	}
	throw new ParseError('vault_kind');
}
