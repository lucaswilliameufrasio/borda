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

export type Device = {
	id: string;
	mac: string;
	address?: string;
	name?: string;
	seenOn?: string;
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
	devices: Device[];
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

export type ProofResult = {
	outcome: 'passed' | 'failed';
	primaryActive: boolean;
	usbReceived: number;
	lanReceived: number;
};

export type ProofBody = {
	confirmed: true;
};

export type WanResult = {
	comment: string;
	interfaceName: string;
	distance: number;
	alreadyPresent: boolean;
};

export type WanBody = {
	name: string;
	interface_name: string;
	distance: number;
	confirmed: true;
};

export type PathReason = 'check_failed' | 'proof_requested' | 'route_restored';

export type PathEvent = {
	id: string;
	at: string;
	reason: PathReason;
};

export type AssistantKind = 'answer' | 'proposal' | 'missing' | 'refused';

export type AssistantProposal = {
	action: 'proof';
	summary: string;
};

export type AssistantReply = {
	kind: AssistantKind;
	text: string;
	missing: string[];
	proposal?: AssistantProposal;
};

export type HubPortrait = {
	session: SessionName;
	hasActiveRoute: boolean;
	activeGateway?: string;
	activeInterface?: string;
	activeDistance?: number;
	leaseCount: number;
	deviceCount: number;
	routeCount: number;
};

export type HubView =
	| { linked: false; name: string }
	| { linked: true; name: string; portrait: HubPortrait; events: PathEvent[] };

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
