<script lang="ts">
	import { resolve } from '$app/paths';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import { m } from '#lib/paraglide/messages.js';
	import { apiBaseUrl } from '#lib/config.ts';
	import { createClient } from '#lib/routers/client.ts';
	import { ApiError, type HubView, type PathReason, type SessionName } from '#lib/routers/model.ts';

	let { data } = $props();

	type HubScreen = { type: 'ready'; view: HubView } | { type: 'submitting'; view: HubView } | { type: 'rejected'; message: string };

	function initialScreen(): HubScreen {
		if (data.state.type !== 'ready') {
			return { type: 'rejected', message: data.state.message };
		}
		return { type: 'ready', view: data.state.view };
	}

	let screen = $state<HubScreen>(initialScreen());

	function errorText(message: string): string {
		if (message === '') {
			return m.loadError();
		}
		return message;
	}

	function sessionText(session: SessionName): string {
		if (session === 'locked') {
			return m.hubSessionLocked();
		}
		if (session === 'unreachable') {
			return m.sessionUnreachable();
		}
		if (session === 'authentication_rejected') {
			return m.sessionRejected();
		}
		return m.hubSessionOpen();
	}

	function reasonText(reason: PathReason): string {
		if (reason === 'check_failed') {
			return m.reasonCheckFailed();
		}
		if (reason === 'proof_requested') {
			return m.reasonProofRequested();
		}
		return m.reasonRouteRestored();
	}

	function formatWhen(value: string): string {
		const parsed = Date.parse(value);
		if (Number.isNaN(parsed)) {
			return value;
		}
		return new Intl.DateTimeFormat(getLocale(), { dateStyle: 'short', timeStyle: 'medium' }).format(parsed);
	}

	function approveLabel(): string {
		if (screen.type === 'submitting') {
			return m.hubApproving();
		}
		return m.hubApprove();
	}

	async function approveAgent() {
		if (screen.type !== 'ready' || screen.view.linked) {
			return;
		}
		if (data.state.type !== 'ready') {
			return;
		}
		const current = screen.view;
		screen = { type: 'submitting', view: current };
		try {
			const view = await createClient(apiBaseUrl(), fetch).approveAgent(data.state.id);
			screen = { type: 'ready', view };
		} catch (error) {
			if (error instanceof ApiError) {
				screen = { type: 'rejected', message: error.message };
				return;
			}
			screen = { type: 'rejected', message: '' };
		}
	}
</script>

{#if data.state.type === 'error'}
	<p class="crumb"><a href={resolve('/')}>{m.back()}</a></p>
	<p class="banner banner-alert">{errorText(data.state.message)}</p>
{:else}
	<p class="crumb">
		<a href={resolve('/routers/[id]', { id: data.state.id })}>{m.back()}</a>
	</p>
	<div class="page-head">
		<div>
			<h2>{m.hubTitle()}</h2>
			<p class="meta">{data.state.view.name}</p>
		</div>
	</div>
	<p class="meta">{m.hubNote()}</p>
	<p class="meta">{m.hubRemote()}</p>
	{#if screen.type === 'rejected'}
		<p class="banner banner-alert">{errorText(screen.message)}</p>
	{:else if screen.view.linked}
		<p class="banner">{sessionText(screen.view.portrait.session)}</p>
		<section class="metrics" aria-label={m.hubTitle()}>
			<article class="metric">
				<span class="kicker">{m.columnRoutes()}</span>
				<strong>{screen.view.portrait.routeCount}</strong>
			</article>
			<article class="metric">
				<span class="kicker">{m.columnLeases()}</span>
				<strong>{screen.view.portrait.leaseCount}</strong>
			</article>
			<article class="metric">
				<span class="kicker">{m.columnDevices()}</span>
				<strong>{screen.view.portrait.deviceCount}</strong>
			</article>
		</section>
		{#if screen.view.portrait.hasActiveRoute && screen.view.portrait.activeGateway !== undefined}
			<p class="meta">
				{m.gateway()}: <span class="mono">{screen.view.portrait.activeGateway}</span>
				{m.egress()}: <span class="mono">{screen.view.portrait.activeInterface ?? ''}</span>
			</p>
		{/if}
		<p class="meta">{m.hubWrite()}</p>
		<section class="panel panel-wide">
			{#if screen.view.events.length === 0}
				<p class="empty">{m.historyEmpty()}</p>
			{:else}
				<div class="table-wrap">
					<table>
						<thead>
							<tr>
								<th>{m.columnWhen()}</th>
								<th>{m.columnReason()}</th>
							</tr>
						</thead>
						<tbody>
							{#each screen.view.events as event (event.id)}
								<tr>
									<td class="mono">{formatWhen(event.at)}</td>
									<td>{reasonText(event.reason)}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>
	{:else}
		<div class="actions">
			<button type="button" disabled={screen.type === 'submitting'} onclick={approveAgent}>{approveLabel()}</button>
		</div>
	{/if}
{/if}
