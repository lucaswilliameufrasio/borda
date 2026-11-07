<script lang="ts">
	import { resolve } from '$app/paths';
	import { apiBaseUrl } from '#lib/config.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { createClient } from '#lib/routers/client.ts';
	import { ApiError } from '#lib/routers/model.ts';

	let { data } = $props();

	type ProofScreen =
		| { type: 'review' }
		| { type: 'running' }
		| { type: 'restored'; outcome: 'passed' | 'failed'; usbReceived: number; lanReceived: number }
		| { type: 'not-restored'; message: string }
		| { type: 'rejected'; message: string };

	let screen = $state<ProofScreen>({ type: 'review' });

	function errorText(message: string): string {
		if (message === '') {
			return m.loadError();
		}
		return message;
	}

	function buttonLabel(): string {
		if (screen.type === 'running') {
			return m.proofRunning();
		}
		return m.proofConfirm();
	}

	async function confirmProof() {
		if (screen.type === 'running') {
			return;
		}
		if (data.state.type !== 'ready') {
			return;
		}
		screen = { type: 'running' };
		try {
			const result = await createClient(apiBaseUrl(), fetch).prove(data.state.router.id);
			if (!result.primaryActive) {
				screen = { type: 'not-restored', message: m.proofNotRestored() };
				return;
			}
			screen = {
				type: 'restored',
				outcome: result.outcome,
				usbReceived: result.usbReceived,
				lanReceived: result.lanReceived,
			};
		} catch (error) {
			if (error instanceof ApiError && error.errorCode === 'PROOF_NOT_RESTORED') {
				screen = { type: 'not-restored', message: error.message };
				return;
			}
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
	<p class="crumb"><a href={resolve('/routers/[id]', { id: data.state.router.id })}>{m.back()}</a></p>
	<div class="page-head">
		<div>
			<h2>{m.proofTitle()}</h2>
			<p class="meta">{data.state.router.name}</p>
		</div>
	</div>
	{#if screen.type === 'not-restored'}
		<p class="banner banner-alert">{errorText(screen.message)}</p>
	{:else if screen.type === 'restored'}
		<p class={screen.outcome === 'passed' ? 'banner' : 'banner banner-locked'}>
			{#if screen.outcome === 'passed'}
				{m.proofPassed()}
			{:else}
				{m.proofFailed()}
			{/if}
		</p>
		<section class="metrics">
			<article class="metric">
				<span class="kicker">{m.proofUsb()}</span>
				<strong>{screen.usbReceived}</strong>
			</article>
			<article class="metric">
				<span class="kicker">{m.proofLan()}</span>
				<strong>{screen.lanReceived}</strong>
			</article>
		</section>
	{:else}
		<p class="banner banner-locked">{m.proofWarning()}</p>
		{#if screen.type === 'rejected'}
			<p class="banner banner-alert">{errorText(screen.message)}</p>
		{/if}
		<button type="button" disabled={screen.type === 'running'} onclick={confirmProof}>{buttonLabel()}</button>
	{/if}
{/if}
