import adapter from '@sveltejs/adapter-auto';
import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [
		sveltekit({ adapter: adapter() }),
		paraglideVitePlugin({
			project: './project.inlang',
			outdir: './src/lib/paraglide',
			emitTsDeclarations: true,
			outputStructure: 'message-modules',
			strategy: ['cookie', 'preferredLanguage', 'baseLocale'],
		}),
	],
	test: {
		include: ['tests/unit/**/*.test.ts'],
		environment: 'node',
	},
});
