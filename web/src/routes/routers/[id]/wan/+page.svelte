<script lang="ts">
	import { resolve } from '$app/paths';
	import { apiBaseUrl } from '#lib/config.ts';
	import { formElement, formString } from '#lib/dom.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { createClient } from '#lib/routers/client.ts';
	import { ApiError, type FieldError } from '#lib/routers/model.ts';
	import { fieldMessage } from '#lib/routers/parse.ts';

	let { data } = $props();

	type WanScreen =
		| { type: 'editing'; notice: string; fieldErrors: FieldError[] }
		| { type: 'submitting' }
		| { type: 'registered'; comment: string; alreadyPresent: boolean };

	let screen = $state<WanScreen>({ type: 'editing', notice: '', fieldErrors: [] });
	let preview = $state('');

	function errorText(message: string): string {
		if (message === '') {
			return m.loadError();
		}
		return message;
	}

	function messageFor(field: string): string | undefined {
		if (screen.type !== 'editing') {
			return undefined;
		}
		return fieldMessage(screen.fieldErrors, field);
	}

	function commentFor(name: string): string {
		if (!/^[A-Za-z][A-Za-z0-9-]{0,23}$/.test(name)) {
			return '';
		}
		return `borda-wan-${name.toLowerCase()}`;
	}

	function distanceFrom(value: string): number | undefined {
		if (!/^[1-9][0-9]{0,2}$/.test(value)) {
			return undefined;
		}
		const parsed = Number(value);
		if (!Number.isInteger(parsed) || parsed < 1 || parsed > 255) {
			return undefined;
		}
		return parsed;
	}

	function buttonLabel(): string {
		if (screen.type === 'submitting') {
			return m.wanSaving();
		}
		return m.wanConfirm();
	}

	function onName(event: Event) {
		const target = event.currentTarget;
		if (!(target instanceof HTMLInputElement)) {
			return;
		}
		preview = commentFor(target.value.trim());
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (screen.type === 'submitting') {
			return;
		}
		if (data.state.type !== 'ready') {
			return;
		}
		const form = formElement(event);
		if (form === undefined) {
			return;
		}
		const name = formString(new FormData(form), 'name').trim();
		const interfaceName = formString(new FormData(form), 'interface_name').trim();
		const distance = distanceFrom(formString(new FormData(form), 'distance').trim());
		if (distance === undefined) {
			screen = {
				type: 'editing',
				notice: '',
				fieldErrors: [{ field: 'distance', message: m.wanDistanceInvalid() }],
			};
			return;
		}
		screen = { type: 'submitting' };
		try {
			const result = await createClient(apiBaseUrl(), fetch).registerWan(data.state.router.id, name, interfaceName, distance);
			screen = { type: 'registered', comment: result.comment, alreadyPresent: result.alreadyPresent };
		} catch (error) {
			if (error instanceof ApiError) {
				let notice = error.message;
				for (const fieldError of error.fieldErrors) {
					if (fieldError.field === 'vault') {
						notice = fieldError.message;
					}
				}
				screen = { type: 'editing', notice, fieldErrors: error.fieldErrors };
				return;
			}
			screen = { type: 'editing', notice: '', fieldErrors: [] };
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
			<h2>{m.wanTitle()}</h2>
			<p class="meta">{data.state.router.name}</p>
		</div>
	</div>
	{#if screen.type === 'registered'}
		<p class="banner">
			{#if screen.alreadyPresent}
				{m.wanAlready()}
			{:else}
				{m.wanRegistered()}
			{/if}
			<span class="mono">{screen.comment}</span>
		</p>
	{:else}
		<p class="banner banner-locked">{m.wanWarning()}</p>
		{#if screen.type === 'editing' && screen.notice !== ''}
			<p class="banner banner-alert">{errorText(screen.notice)}</p>
		{/if}
		<form class="setup" onsubmit={submit}>
			<label>
				{m.wanName()}
				<input name="name" autocomplete="off" oninput={onName} />
				{#if preview !== ''}
					<span class="hint">{m.wanCommentLabel()} <span class="mono">{preview}</span></span>
				{/if}
				{#if messageFor('name') !== undefined}
					<span class="error">{messageFor('name')}</span>
				{/if}
			</label>
			<label>
				{m.wanInterface()}
				<input name="interface_name" autocomplete="off" />
				{#if messageFor('interface_name') !== undefined}
					<span class="error">{messageFor('interface_name')}</span>
				{/if}
			</label>
			<label>
				{m.wanDistance()}
				<input name="distance" inputmode="numeric" autocomplete="off" />
				{#if messageFor('distance') !== undefined}
					<span class="error">{messageFor('distance')}</span>
				{/if}
			</label>
			<div class="actions">
				<button type="submit" disabled={screen.type === 'submitting'}>{buttonLabel()}</button>
			</div>
		</form>
	{/if}
{/if}
