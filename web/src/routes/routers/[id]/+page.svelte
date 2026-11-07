<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { apiBaseUrl } from '#lib/config.ts';
	import { formElement, formString } from '#lib/dom.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { createClient } from '#lib/routers/client.ts';
	import { ApiError, type Snapshot } from '#lib/routers/model.ts';
	import { fieldMessage } from '#lib/routers/parse.ts';

	let { data } = $props();

	type UnlockState = { type: 'idle' } | { type: 'submitting' } | { type: 'rejected'; message: string };
	let unlock = $state<UnlockState>({ type: 'idle' });

	function sessionText(session: 'unreachable' | 'authentication_rejected' | 'locked'): string {
		if (session === 'unreachable') {
			return m.sessionUnreachable();
		}
		if (session === 'authentication_rejected') {
			return m.sessionRejected();
		}
		return m.sessionLocked();
	}

	function errorText(message: string): string {
		if (message === '') {
			return m.loadError();
		}
		return message;
	}

	function activeLabel(active: boolean): string {
		if (active) {
			return m.activeYes();
		}
		return m.activeNo();
	}

	function portName(name: string | undefined): string {
		if (name === undefined || name === '') {
			return m.emptyInterface();
		}
		return name;
	}

	function sessionClass(session: 'unreachable' | 'authentication_rejected' | 'locked'): string {
		if (session === 'locked') {
			return 'banner banner-locked';
		}
		return 'banner banner-alert';
	}

	async function submitUnlock(event: SubmitEvent) {
		event.preventDefault();
		if (unlock.type === 'submitting') {
			return;
		}
		if (data.state.type !== 'ready' || data.state.view.type !== 'session-failed') {
			return;
		}
		const form = formElement(event);
		if (form === undefined) {
			return;
		}
		const formData = new FormData(form);
		unlock = { type: 'submitting' };
		try {
			await createClient(apiBaseUrl(), fetch).unlock(
				data.state.router.id,
				formString(formData, 'vault_passphrase'),
				formString(formData, 'access_token'),
			);
			unlock = { type: 'idle' };
			await invalidateAll();
		} catch (error) {
			if (error instanceof ApiError) {
				const vaultMessage = fieldMessage(error.fieldErrors, 'vault');
				if (vaultMessage !== undefined) {
					unlock = { type: 'rejected', message: vaultMessage };
					return;
				}
				unlock = { type: 'rejected', message: error.message };
				return;
			}
			unlock = { type: 'rejected', message: '' };
		}
	}
</script>

<p class="crumb"><a href={resolve('/')}>{m.back()}</a></p>

{#if data.state.type === 'error'}
	<p class="banner banner-alert">{errorText(data.state.message)}</p>
{:else if data.state.view.type === 'session-failed'}
	{@const failed = data.state.view}
	{@const router = data.state.router}
	<div class="page-head">
		<h2>{router.name}</h2>
	</div>
	<p class={sessionClass(failed.session)}>{sessionText(failed.session)}</p>
	{#if failed.session === 'locked' && (router.vaultKind === 'file' || router.vaultKind === 'ciphera' || router.vaultKind === 'infisical')}
		<form class="unlock" onsubmit={submitUnlock}>
			{#if router.vaultKind === 'file'}
				<label>
					{m.fieldVaultPassphrase()}
					<input name="vault_passphrase" type="password" autocomplete="new-password" />
				</label>
			{:else}
				<label>
					{m.fieldAccessToken()}
					<input name="access_token" type="password" autocomplete="new-password" />
				</label>
			{/if}
			{#if unlock.type === 'rejected'}
				<p class="error">{errorText(unlock.message)}</p>
			{/if}
			<div class="actions">
				<button type="submit" disabled={unlock.type === 'submitting'}>{m.unlock()}</button>
			</div>
		</form>
	{/if}
{:else}
	{@const snapshot = data.state.view.snapshot}
	<div class="page-head">
		<div>
			<h2>{data.state.router.name}</h2>
			{#if snapshot.identity !== undefined}
				<p class="meta">
					{m.identity()}:
					<span class="mono">{snapshot.identity.name ?? ''} {snapshot.identity.version ?? ''}</span>
				</p>
			{/if}
		</div>
		<a class="button button-quiet" href={resolve('/routers/[id]/proof', { id: data.state.router.id })}>{m.proofNav()}</a>
	</div>
	{#if data.state.view.type === 'empty-route'}
		<p class="banner banner-locked">{m.noDefaultRoute()}</p>
	{:else if snapshot.activeRoute !== undefined}
		{@const route = snapshot.activeRoute}
		<section class="metrics" aria-label={m.activeRoute()}>
			<article class="metric">
				<span class="kicker">{m.gateway()}</span>
				<strong>{route.gateway}</strong>
			</article>
			<article class="metric">
				<span class="kicker">{m.egress()}</span>
				<strong>{portName(route.interfaceName)}</strong>
			</article>
			<article class="metric">
				<span class="kicker">{m.distance()}</span>
				<strong>{route.distance}</strong>
			</article>
		</section>
	{/if}
	<div class="status-row">
		{#if snapshot.check !== undefined}
			<p class={snapshot.check.responding ? 'pill pill-ok' : 'pill'}>
				<span>
					{#if snapshot.check.responding}
						{m.checkResponding()}
					{:else}
						{m.checkSilent()}
					{/if}
				</span>
				<span class="mono">{snapshot.check.address}</span>
			</p>
		{/if}
		{#if snapshot.reserve !== undefined}
			<p class="pill">{m.reserveReady()} <span class="mono">{snapshot.reserve.interfaceName}</span></p>
		{/if}
	</div>
	<div class="panels">
		{@render tables(snapshot, data.state.router.id)}
	</div>
{/if}

{#snippet tables(snapshot: Snapshot, routerId: string)}
	<section class="panel">
		<h2>{m.routesTitle()}</h2>
		{#if snapshot.routes.length === 0}
			<p class="empty">{m.noRoutes()}</p>
		{:else}
			<div class="table-wrap">
				<table>
					<thead>
						<tr>
							<th>{m.gateway()}</th>
							<th>{m.egress()}</th>
							<th>{m.distance()}</th>
							<th>{m.activeRoute()}</th>
						</tr>
					</thead>
					<tbody>
						{#each snapshot.routes as route, index (`${index}-${route.gateway}`)}
							<tr class:active={route.active}>
								<td>{route.gateway}</td>
								<td>{portName(route.interfaceName)}</td>
								<td>{route.distance}</td>
								<td>{activeLabel(route.active)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>

	<section class="panel">
		<h2>{m.forwardTitle()}</h2>
		{#if snapshot.forwardRules.length === 0}
			<p class="empty">{m.noForward()}</p>
		{:else}
			<div class="table-wrap">
				<table>
					<thead>
						<tr>
							<th>{m.columnAction()}</th>
							<th>{m.columnComment()}</th>
							<th>{m.columnIn()}</th>
							<th>{m.columnOut()}</th>
						</tr>
					</thead>
					<tbody>
						{#each snapshot.forwardRules as rule, index (`${index}-${rule.action}`)}
							<tr>
								<td>{rule.action}</td>
								<td>{rule.comment ?? ''}</td>
								<td>{rule.inInterface ?? ''}</td>
								<td>{rule.outInterface ?? ''}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>

	<section class="panel panel-wide">
		<h2>{m.devicesTitle()}</h2>
		{#if snapshot.devices.length === 0}
			<p class="empty">{m.noDevices()}</p>
		{:else}
			<div class="table-wrap">
				<table>
					<thead>
						<tr>
							<th>{m.columnHostName()}</th>
							<th>{m.columnAddress()}</th>
							<th>{m.columnMac()}</th>
							<th>{m.columnSeenOn()}</th>
						</tr>
					</thead>
					<tbody>
						{#each snapshot.devices as device (device.id)}
							<tr>
								<td>
									<a class="row-link" href={resolve('/routers/[id]/devices/[mac]', { id: routerId, mac: device.id })}>
										{device.name ?? device.address ?? device.mac}
									</a>
								</td>
								<td>{device.address ?? ''}</td>
								<td class="mono">{device.mac}</td>
								<td>{device.seenOn ?? ''}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>

	<section class="panel panel-wide">
		<h2>{m.leasesTitle()}</h2>
		{#if snapshot.leases.length === 0}
			<p class="empty">{m.noLeases()}</p>
		{:else}
			<div class="table-wrap">
				<table>
					<thead>
						<tr>
							<th>{m.columnAddress()}</th>
							<th>{m.columnMac()}</th>
							<th>{m.columnHostName()}</th>
						</tr>
					</thead>
					<tbody>
						{#each snapshot.leases as lease (`${lease.mac}-${lease.address}`)}
							<tr>
								<td>{lease.address}</td>
								<td>{lease.mac}</td>
								<td>{lease.hostName ?? ''}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>
{/snippet}
