<script lang="ts">
	import { resolve } from '$app/paths';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import { m } from '#lib/paraglide/messages.js';
	import type { PathReason } from '#lib/routers/model.ts';

	let { data } = $props();

	function errorText(message: string): string {
		if (message === '') {
			return m.loadError();
		}
		return message;
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
			<h2>{m.historyTitle()}</h2>
			<p class="meta">{data.state.router.name}</p>
		</div>
	</div>
	<p class="meta">{m.historyNote()}</p>
	<section class="panel panel-wide">
		{#if data.state.events.length === 0}
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
						{#each data.state.events as event (event.id)}
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
{/if}
