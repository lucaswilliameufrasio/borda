import { defineEnvVars } from '@sveltejs/kit/env';

export const variables = defineEnvVars({
	PUBLIC_BORDA_API_URL: {
		public: true,
		schema: (value: string | undefined) => {
			if (value === undefined || value === '') {
				return 'http://127.0.0.1:8787';
			}
			return value;
		},
	},
});
