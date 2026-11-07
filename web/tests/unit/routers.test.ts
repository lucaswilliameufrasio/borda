import { expect, it } from 'vitest';
import { ParseError } from '../../src/lib/routers/model';
import { detailView, fieldMessage, findDevice, parseEvents, parseProof, parseSnapshot } from '../../src/lib/routers/parse';
import { buildProofBody, buildRegisterBody } from '../../src/lib/routers/request';

it('Should show an empty route when no default is active', () => {
	const snapshot = parseSnapshot({
		router_id: 'empty-router',
		name: 'Vazio',
		session: 'connected',
		routes: [],
		forward_rules: [],
		leases: [],
	});
	const view = detailView(snapshot);
	expect(view.type).toBe('empty-route');
	if (view.type === 'empty-route') {
		expect(view.snapshot.leases).toEqual([]);
		expect(view.snapshot.activeRoute).toBeUndefined();
	}
});

it('Should show the active path and leases without keeping a password', () => {
	const snapshot = parseSnapshot({
		router_id: 'edge-router',
		name: 'Casa',
		session: 'connected',
		router_password: 'should-not-leak',
		active_route: { gateway: '192.0.2.1', interface_name: 'ether1', distance: 1 },
		check: { address: '192.0.2.1', responding: true },
		routes: [{ gateway: '192.0.2.1', active: true, distance: 1, interface_name: 'ether1' }],
		forward_rules: [{ action: 'accept', comment: 'lan', in_interface: 'bridge', out_interface: 'ether1' }],
		leases: [{ address: '192.0.2.20', mac: 'AA:BB:CC:DD:EE:FF', host_name: 'phone' }],
	});
	expect(JSON.stringify(snapshot).includes('should-not-leak')).toBe(false);
	const view = detailView(snapshot);
	expect(view.type).toBe('connected');
	if (view.type === 'connected') {
		expect(view.snapshot.activeRoute?.interfaceName).toBe('ether1');
		expect(view.snapshot.check?.responding).toBe(true);
		expect(view.snapshot.leases[0]?.hostName).toBe('phone');
	}
});

it('Should map a failed session without inventing a path', () => {
	const snapshot = parseSnapshot({
		router_id: 'offline-router',
		name: 'Fora',
		session: 'unreachable',
	});
	const view = detailView(snapshot);
	expect(view.type).toBe('session-failed');
	if (view.type === 'session-failed') {
		expect(view.session).toBe('unreachable');
	}
	expect(snapshot.routes).toEqual([]);
	expect(snapshot.activeRoute).toBeUndefined();
});

it('Should not share lease arrays between snapshots', () => {
	const first = parseSnapshot({
		router_id: 'one',
		name: 'Um',
		session: 'connected',
		leases: [{ address: '192.0.2.20', mac: 'AA:BB:CC:DD:EE:FF', host_name: 'phone' }],
	});
	const second = parseSnapshot({
		router_id: 'two',
		name: 'Dois',
		session: 'connected',
		leases: [],
	});
	first.leases.push({ address: '192.0.2.21', mac: 'AA:BB:CC:DD:EE:00' });
	expect(second.leases).toEqual([]);
	expect(second.name).toBe('Dois');
});

it('Should find a device by mac id', () => {
	const snapshot = parseSnapshot({
		router_id: 'edge-router',
		name: 'Casa',
		session: 'connected',
		devices: [
			{
				id: 'aabbccddeeff',
				mac: 'AA:BB:CC:DD:EE:FF',
				address: '192.0.2.20',
				name: 'phone',
				seen_on: 'bridge',
			},
		],
	});
	expect(findDevice(snapshot.devices, 'aabbccddeeff')?.name).toBe('phone');
	expect(findDevice(snapshot.devices, 'aabbccddeeff')?.seenOn).toBe('bridge');
	expect(findDevice(snapshot.devices, '000000000000')).toBeUndefined();
	expect(JSON.stringify(snapshot).includes('router_password')).toBe(false);
});

it('Should keep path events in the order the hub returned', () => {
	const events = parseEvents([
		{ id: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', at: '2026-11-07T15:04:00Z', reason: 'route_restored' },
		{ id: 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb', at: '2026-11-07T15:03:00Z', reason: 'proof_requested' },
	]);
	expect(events.map((event) => event.reason)).toEqual(['route_restored', 'proof_requested']);
	expect(parseEvents([])).toEqual([]);
	expect(() => parseEvents([{ id: 'cccccccccccccccccccccccccccccccc', at: '2026-11-07T15:00:00Z', reason: 'guessed' }])).toThrow(
		ParseError,
	);
});

it('Should build a file vault body without a remote token', () => {
	const body = buildRegisterBody({
		name: 'Casa',
		host: 'connected.example',
		sshPort: '',
		username: 'reader',
		checkAddress: '192.0.2.1',
		vault: { type: 'file', routerPassword: 'router-demo', vaultPassphrase: 'vault-demo' },
	});
	expect(body.vault_kind).toBe('file');
	expect(body.ssh_port).toBeUndefined();
	expect(body.access_token).toBeUndefined();
	expect(body.check_address).toBe('192.0.2.1');
	expect(Object.hasOwn(body, 'username')).toBe(true);
});

it('Should read a field message by name', () => {
	expect(fieldMessage([{ field: 'host', message: 'Informe o endereço do roteador.' }], 'host')).toBe(
		'Informe o endereço do roteador.',
	);
	expect(fieldMessage([], 'host')).toBeUndefined();
});

it('Should reject a snapshot that is not an object', () => {
	expect(() => parseSnapshot([])).toThrow(ParseError);
});

it('Should build a confirmed proof without a secret field', () => {
	const body = buildProofBody();
	expect(body.confirmed).toBe(true);
	expect(Object.hasOwn(body, 'router_password')).toBe(false);
	expect(Object.hasOwn(body, 'vault_passphrase')).toBe(false);
});

it('Should read a restored proof that did not pass', () => {
	const proof = parseProof({
		outcome: 'failed',
		primary_active: true,
		usb_received: 0,
		lan_received: 1,
	});
	expect(proof.outcome).toBe('failed');
	expect(proof.primaryActive).toBe(true);
	expect(proof.usbReceived).toBe(0);
	expect(proof.lanReceived).toBe(1);
});
