<script lang="ts">
	import { resolve } from '$app/paths';
	import { apiBaseUrl } from '#lib/config.ts';
	import { formElement, formString } from '#lib/dom.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { createClient } from '#lib/routers/client.ts';
	import { ApiError, type AssistantReply } from '#lib/routers/model.ts';

	let { data } = $props();

	type AskScreen =
		| { type: 'idle' }
		| { type: 'submitting' }
		| { type: 'result'; reply: AssistantReply }
		| { type: 'running'; reply: AssistantReply }
		| { type: 'done'; outcome: 'passed' | 'failed' }
		| { type: 'rejected'; message: string };

	let screen = $state<AskScreen>({ type: 'idle' });

	function errorText(message: string): string {
		if (message === '') {
			return m.loadError();
		}
		return message;
	}

	function sendLabel(): string {
		if (screen.type === 'submitting') {
			return m.assistantSending();
		}
		return m.assistantSend();
	}

	function confirmLabel(): string {
		if (screen.type === 'running') {
			return m.assistantConfirming();
		}
		return m.assistantConfirm();
	}

	function outcomeText(outcome: 'passed' | 'failed'): string {
		if (outcome === 'passed') {
			return m.proofPassed();
		}
		return m.proofFailed();
	}

	async function submitQuestion(event: SubmitEvent) {
		event.preventDefault();
		if (screen.type === 'submitting' || screen.type === 'running') {
			return;
		}
		if (data.state.type !== 'ready') {
			return;
		}
		const form = formElement(event);
		if (form === undefined) {
			return;
		}
		const question = formString(new FormData(form), 'question');
		screen = { type: 'submitting' };
		try {
			const reply = await createClient(apiBaseUrl(), fetch).ask(data.state.router.id, question);
			screen = { type: 'result', reply };
		} catch (error) {
			if (error instanceof ApiError) {
				screen = { type: 'rejected', message: error.message };
				return;
			}
			screen = { type: 'rejected', message: '' };
		}
	}

	async function confirmProposal() {
		if (screen.type !== 'result' || screen.reply.kind !== 'proposal') {
			return;
		}
		if (data.state.type !== 'ready') {
			return;
		}
		const reply = screen.reply;
		screen = { type: 'running', reply };
		try {
			const result = await createClient(apiBaseUrl(), fetch).applyProposal(data.state.router.id);
			if (!result.primaryActive) {
				screen = { type: 'rejected', message: m.proofNotRestored() };
				return;
			}
			screen = { type: 'done', outcome: result.outcome };
		} catch (error) {
			if (error instanceof ApiError && error.errorCode === 'PROOF_NOT_RESTORED') {
				screen = { type: 'rejected', message: error.message };
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
	<p class="crumb">
		<a href={resolve('/routers/[id]', { id: data.state.router.id })}>{m.back()}</a>
	</p>
	<div class="page-head">
		<div>
			<h2>{m.assistantTitle()}</h2>
			<p class="meta">{data.state.router.name}</p>
		</div>
	</div>
	<p class="meta">{m.assistantNote()}</p>
	<form class="panel panel-wide" onsubmit={submitQuestion}>
		<label>
			{m.assistantQuestion()}
			<textarea class="question" name="question" rows="4"></textarea>
		</label>
		<div class="actions">
			<button type="submit" disabled={screen.type === 'submitting' || screen.type === 'running'}>{sendLabel()}</button>
		</div>
	</form>
	{#if screen.type === 'rejected'}
		<p class="banner banner-alert">{errorText(screen.message)}</p>
	{:else if screen.type === 'done'}
		<p class="banner">{outcomeText(screen.outcome)}</p>
	{:else if screen.type === 'result' || screen.type === 'running'}
		<section class="panel panel-wide">
			<p>{screen.reply.text}</p>
			{#if screen.reply.kind === 'proposal' && screen.reply.proposal !== undefined}
				<p>{screen.reply.proposal.summary}</p>
				<div class="actions">
					<button type="button" disabled={screen.type === 'running'} onclick={confirmProposal}>{confirmLabel()}</button>
				</div>
			{/if}
		</section>
	{/if}
{/if}
