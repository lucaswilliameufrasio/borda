import type { RegisterBody, SetupDraft } from './model';

export function buildRegisterBody(draft: SetupDraft): RegisterBody {
	const body: RegisterBody = {
		name: draft.name.trim(),
		host: draft.host.trim(),
		username: draft.username.trim(),
		vault_kind: draft.vault.type,
	};
	if (draft.sshPort.trim() !== '') {
		const port = Number(draft.sshPort);
		if (Number.isInteger(port)) {
			body.ssh_port = port;
		}
	}
	if (draft.checkAddress.trim() !== '') {
		body.check_address = draft.checkAddress.trim();
	}
	if (draft.vault.type === 'file') {
		body.router_password = draft.vault.routerPassword;
		body.vault_passphrase = draft.vault.vaultPassphrase;
	}
	if (draft.vault.type === 'keychain') {
		body.router_password = draft.vault.routerPassword;
	}
	if (draft.vault.type === 'ciphera') {
		body.api_url = draft.vault.apiUrl.trim();
		body.project_id = draft.vault.projectId.trim();
		body.environment = draft.vault.environment.trim();
		body.secret_key = draft.vault.secretKey.trim();
		body.access_token = draft.vault.accessToken;
	}
	if (draft.vault.type === 'infisical') {
		body.api_url = draft.vault.apiUrl.trim();
		body.project_id = draft.vault.projectId.trim();
		body.environment = draft.vault.environment.trim();
		body.secret_name = draft.vault.secretName.trim();
		body.access_token = draft.vault.accessToken;
	}
	return body;
}

export function buildUnlockBody(passphrase: string, token: string): RegisterBody {
	const body: RegisterBody = {};
	if (passphrase !== '') {
		body.vault_passphrase = passphrase;
	}
	if (token !== '') {
		body.access_token = token;
	}
	return body;
}
