export type VaultKind = 'file' | 'keychain' | 'ciphera' | 'infisical';

export type SessionName = 'connected' | 'unreachable' | 'authentication_rejected' | 'locked';

export type RouterSummary = {
	id: string;
	name: string;
	host: string;
	sshPort: number;
	checkAddress?: string;
	vaultKind: VaultKind;
};

export type ActiveRoute = {
	gateway: string;
	interfaceName?: string;
	distance: number;
};

export type CheckResult = {
	address: string;
	responding: boolean;
};

export type RouteRow = {
	gateway: string;
	active: boolean;
	distance: number;
	interfaceName?: string;
};

export type ForwardRule = {
	action: string;
	comment?: string;
	inInterface?: string;
	outInterface?: string;
};

export type Lease = {
	address: string;
	mac: string;
	hostName?: string;
};

export type Reserve = {
	interfaceName: string;
	running: boolean;
	hasAddress: boolean;
};

export type Snapshot = {
	routerId: string;
	name: string;
	session: SessionName;
	identity?: { name?: string; version?: string };
	activeRoute?: ActiveRoute;
	check?: CheckResult;
	routes: RouteRow[];
	forwardRules: ForwardRule[];
	leases: Lease[];
	reserve?: Reserve;
};

export type DetailView =
	| { type: 'empty-route'; snapshot: Snapshot }
	| { type: 'connected'; snapshot: Snapshot }
	| { type: 'session-failed'; session: 'unreachable' | 'authentication_rejected' | 'locked' };

export type FieldError = {
	field: string;
	message: string;
};

export type VaultForm =
	| { type: 'file'; routerPassword: string; vaultPassphrase: string }
	| { type: 'keychain'; routerPassword: string }
	| {
			type: 'ciphera';
			apiUrl: string;
			projectId: string;
			environment: string;
			secretKey: string;
			accessToken: string;
	  }
	| {
			type: 'infisical';
			apiUrl: string;
			projectId: string;
			environment: string;
			secretName: string;
			accessToken: string;
	  };

export type SetupDraft = {
	name: string;
	host: string;
	sshPort: string;
	username: string;
	checkAddress: string;
	vault: VaultForm;
};

export type RegisterBody = Record<string, string | number>;

export class ParseError extends Error {
	readonly field: string;

	constructor(field: string) {
		super(`invalid ${field}`);
		this.field = field;
	}
}

export class ApiError extends Error {
	readonly errorCode: string;
	readonly fieldErrors: FieldError[];

	constructor(message: string, errorCode: string, fieldErrors: FieldError[]) {
		super(message);
		this.errorCode = errorCode;
		this.fieldErrors = fieldErrors;
	}
}
