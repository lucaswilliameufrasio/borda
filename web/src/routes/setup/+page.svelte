<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { apiBaseUrl } from '#lib/config.ts';
	import { formElement, formString } from '#lib/dom.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { createClient } from '#lib/routers/client.ts';
	import { ApiError, type FieldError, type SetupDraft, type VaultForm } from '#lib/routers/model.ts';
	import { fieldMessage } from '#lib/routers/parse.ts';

	type VaultChoice = { type: 'file' } | { type: 'keychain' } | { type: 'ciphera' } | { type: 'infisical' };
	type Screen =
		| { type: 'editing'; notice: string; fieldErrors: FieldError[] }
		| { type: 'submitting' };

	let vault = $state<VaultChoice>({ type: 'file' });
	let screen = $state<Screen>({ type: 'editing', notice: '', fieldErrors: [] });

	function chooseVault(choice: VaultChoice) {
		vault = choice;
	}

	function buttonLabel(): string {
		if (screen.type === 'submitting') {
			return m.saving();
		}
		return m.save();
	}

	function messageFor(field: string): string | undefined {
		if (screen.type !== 'editing') {
			return undefined;
		}
		return fieldMessage(screen.fieldErrors, field);
	}

	function draftFrom(form: HTMLFormElement): SetupDraft {
		const formData = new FormData(form);
		let selected: VaultForm = {
			type: 'file',
			routerPassword: formString(formData, 'router_password'),
			vaultPassphrase: formString(formData, 'vault_passphrase'),
		};
		if (vault.type === 'keychain') {
			selected = { type: 'keychain', routerPassword: formString(formData, 'router_password') };
		}
		if (vault.type === 'ciphera') {
			selected = {
				type: 'ciphera',
				apiUrl: formString(formData, 'api_url'),
				projectId: formString(formData, 'project_id'),
				environment: formString(formData, 'environment'),
				secretKey: formString(formData, 'secret_key'),
				accessToken: formString(formData, 'access_token'),
			};
		}
		if (vault.type === 'infisical') {
			selected = {
				type: 'infisical',
				apiUrl: formString(formData, 'api_url'),
				projectId: formString(formData, 'project_id'),
				environment: formString(formData, 'environment'),
				secretName: formString(formData, 'secret_name'),
				accessToken: formString(formData, 'access_token'),
			};
		}
		return {
			name: formString(formData, 'name'),
			host: formString(formData, 'host'),
			sshPort: formString(formData, 'ssh_port'),
			username: formString(formData, 'username'),
			checkAddress: formString(formData, 'check_address'),
			vault: selected,
		};
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (screen.type === 'submitting') {
			return;
		}
		const form = formElement(event);
		if (form === undefined) {
			return;
		}
		screen = { type: 'submitting' };
		try {
			const created = await createClient(apiBaseUrl(), fetch).createRouter(draftFrom(form));
			await goto(resolve('/routers/[id]', { id: created.id }));
		} catch (error) {
			if (error instanceof ApiError) {
				screen = { type: 'editing', notice: error.message, fieldErrors: error.fieldErrors };
				return;
			}
			screen = { type: 'editing', notice: '', fieldErrors: [] };
		}
	}
</script>

<p class="crumb"><a href={resolve('/')}>{m.back()}</a></p>
<div class="page-head">
	<h2>{m.setupTitle()}</h2>
</div>

<form class="setup" onsubmit={submit}>
	<div class="form-grid">
		<section class="fields">
			<label>
				{m.fieldName()}
				<input name="name" autocomplete="off" />
				{#if messageFor('name') !== undefined}
					<span class="error">{messageFor('name')}</span>
				{/if}
			</label>
			<label>
				{m.fieldHost()}
				<input name="host" autocomplete="off" />
				{#if messageFor('host') !== undefined}
					<span class="error">{messageFor('host')}</span>
				{/if}
			</label>
			<label>
				{m.fieldSshPort()}
				<input name="ssh_port" inputmode="numeric" autocomplete="off" />
				<span class="hint">{m.sshPortHint()}</span>
				{#if messageFor('ssh_port') !== undefined}
					<span class="error">{messageFor('ssh_port')}</span>
				{/if}
			</label>
			<label>
				{m.fieldUsername()}
				<input name="username" autocomplete="off" />
				{#if messageFor('username') !== undefined}
					<span class="error">{messageFor('username')}</span>
				{/if}
			</label>
			<label>
				{m.fieldCheck()}
				<input name="check_address" autocomplete="off" placeholder={m.checkPlaceholder()} />
				<span class="hint">{m.checkHint()}</span>
				{#if messageFor('check_address') !== undefined}
					<span class="error">{messageFor('check_address')}</span>
				{/if}
			</label>
		</section>

		<fieldset class="vault">
			<legend>{m.vaultKind()}</legend>
			<div class="choices">
				<label class="choice">
					<input
						type="radio"
						name="vault_kind"
						checked={vault.type === 'file'}
						onchange={() => chooseVault({ type: 'file' })}
					/>
					{m.vaultFile()}
				</label>
				<label class="choice">
					<input
						type="radio"
						name="vault_kind"
						checked={vault.type === 'keychain'}
						onchange={() => chooseVault({ type: 'keychain' })}
					/>
					{m.vaultKeychain()}
				</label>
				<label class="choice">
					<input
						type="radio"
						name="vault_kind"
						checked={vault.type === 'ciphera'}
						onchange={() => chooseVault({ type: 'ciphera' })}
					/>
					{m.vaultCiphera()}
				</label>
				<label class="choice">
					<input
						type="radio"
						name="vault_kind"
						checked={vault.type === 'infisical'}
						onchange={() => chooseVault({ type: 'infisical' })}
					/>
					{m.vaultInfisical()}
				</label>
			</div>
			{#if messageFor('vault_kind') !== undefined}
				<p class="error">{messageFor('vault_kind')}</p>
			{/if}

			{#if vault.type === 'file'}
				<label>
					{m.fieldRouterPassword()}
					<input name="router_password" type="password" autocomplete="new-password" />
				</label>
				<label>
					{m.fieldVaultPassphrase()}
					<input name="vault_passphrase" type="password" autocomplete="new-password" />
				</label>
			{/if}
			{#if vault.type === 'keychain'}
				<label>
					{m.fieldRouterPassword()}
					<input name="router_password" type="password" autocomplete="new-password" />
				</label>
			{/if}
			{#if vault.type === 'ciphera' || vault.type === 'infisical'}
				<label>
					{m.fieldApiUrl()}
					<input name="api_url" autocomplete="off" />
				</label>
				<label>
					{m.fieldProject()}
					<input name="project_id" autocomplete="off" />
				</label>
				<label>
					{m.fieldEnvironment()}
					<input name="environment" autocomplete="off" />
				</label>
				{#if vault.type === 'ciphera'}
					<label>
						{m.fieldSecretKey()}
						<input name="secret_key" autocomplete="off" />
					</label>
				{/if}
				{#if vault.type === 'infisical'}
					<label>
						{m.fieldSecretName()}
						<input name="secret_name" autocomplete="off" />
					</label>
				{/if}
				<label>
					{m.fieldAccessToken()}
					<input name="access_token" type="password" autocomplete="new-password" />
				</label>
			{/if}
			{#if messageFor('vault') !== undefined}
				<p class="error">{messageFor('vault')}</p>
			{/if}
			{#if screen.type === 'editing' && screen.notice !== ''}
				<p class="banner banner-alert">{screen.notice}</p>
			{/if}
			<div class="actions">
				<button type="submit" disabled={screen.type === 'submitting'}>{buttonLabel()}</button>
			</div>
		</fieldset>
	</div>
</form>
