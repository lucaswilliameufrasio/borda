<script lang="ts">
	import { resolve } from '$app/paths';
	import { m } from '#lib/paraglide/messages.js';

	let { data } = $props();

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

	function portName(name: string | undefined): string {
		if (name === undefined || name === '') {
			return m.emptyInterface();
		}
		return name;
	}

	function presenceText(presence: 'now' | 'recent' | 'absent' | undefined): string {
		if (presence === 'recent') {
			return m.presenceRecent();
		}
		if (presence === 'absent') {
			return m.presenceAbsent();
		}
		if (presence === 'now') {
			return m.presenceNow();
		}
		return '';
	}

	function sessionClass(session: 'unreachable' | 'authentication_rejected' | 'locked'): string {
		if (session === 'locked') {
			return 'banner banner-locked';
		}
		return 'banner banner-alert';
	}
</script>

{#if data.state.type === 'error'}
	<p class="crumb"><a href={resolve('/')}>{m.back()}</a></p>
	<p class="banner banner-alert">{errorText(data.state.message)}</p>
{:else}
	<p class="crumb">
		<a href={resolve('/routers/[id]', { id: data.state.router.id })}>{m.back()}</a>
	</p>
	{#if data.state.type === 'session-failed'}
		<div class="page-head">
			<h2>{data.state.router.name}</h2>
		</div>
		<p class={sessionClass(data.state.session)}>{sessionText(data.state.session)}</p>
	{:else if data.state.type === 'missing'}
		<div class="page-head">
			<h2>{data.state.router.name}</h2>
		</div>
		<p class="banner banner-locked">{m.deviceMissing()}</p>
	{:else}
		{@const device = data.state.device}
		<div class="page-head">
			<div>
				<h2>{device.name ?? m.deviceNoName()}</h2>
				<p class="meta"><span class="mono">{device.mac}</span></p>
			</div>
		</div>
		<section class="metrics" aria-label={m.devicesTitle()}>
			<article class="metric">
				<span class="kicker">{m.columnAddress()}</span>
				<strong>{device.address ?? ''}</strong>
			</article>
			<article class="metric">
				<span class="kicker">{m.columnSeenOn()}</span>
				<strong>{portName(device.seenOn)}</strong>
			</article>
			<article class="metric">
				<span class="kicker">{m.columnPresence()}</span>
				<strong>{presenceText(device.presence)}</strong>
			</article>
		</section>
		<section class="panel">
			<h2>{m.deviceExit()}</h2>
			<p class="meta">{m.deviceExitNote()}</p>
			{#if data.state.activeRoute === undefined}
				<p class="empty">{m.noDefaultRoute()}</p>
			{:else}
				{@const route = data.state.activeRoute}
				<section class="metrics" aria-label={m.deviceExit()}>
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
		</section>
	{/if}
{/if}
