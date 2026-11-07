<script lang="ts">
	import { resolve } from '$app/paths';
	import { m } from '#lib/paraglide/messages.js';
	import type { VaultKind } from '#lib/routers/model.ts';

	let { data } = $props();

	function vaultLabel(kind: VaultKind): string {
		if (kind === 'file') {
			return m.vaultFile();
		}
		if (kind === 'keychain') {
			return m.vaultKeychain();
		}
		if (kind === 'ciphera') {
			return m.vaultCiphera();
		}
		return m.vaultInfisical();
	}

	function errorText(message: string): string {
		if (message === '') {
			return m.loadError();
		}
		return message;
	}
</script>

<div class="toolbar">
	<h2>{m.listTitle()}</h2>
	<a class="button" href={resolve('/setup')}>{m.navSetup()}</a>
</div>

{#if data.page.type === 'error'}
	<p class="banner banner-alert">{errorText(data.page.message)}</p>
{:else if data.page.routers.length === 0}
	<p class="blank">{m.emptyList()}</p>
{:else}
	<ul class="router-list">
		{#each data.page.routers as router (router.id)}
			<li>
				<a class="router-card" href={resolve('/routers/[id]', { id: router.id })}>
					<span class="router-id">
						<strong class="router-name">{router.name}</strong>
						<span class="host mono">{router.host}</span>
						<span class="pill">{m.vaultKind()}: {vaultLabel(router.vaultKind)}</span>
					</span>
					<span class="open">{m.openRouter()}</span>
				</a>
			</li>
		{/each}
	</ul>
{/if}
