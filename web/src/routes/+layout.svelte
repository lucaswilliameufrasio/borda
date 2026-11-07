<script lang="ts">
	import { resolve } from '$app/paths';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale, setLocale } from '#lib/paraglide/runtime.js';
	import '../app.css';

	let { children } = $props();

	function chooseLocale(locale: 'pt-BR' | 'en') {
		setLocale(locale);
	}

	function pressed(locale: 'pt-BR' | 'en'): boolean {
		return getLocale() === locale;
	}
</script>

<div class="shell">
	<header class="topbar">
		<a class="brand" href={resolve('/')}>
			<span class="mark" aria-hidden="true"></span>
			<span>
				<h1>{m.appTitle()}</h1>
				<span class="brand-tag">{m.tagline()}</span>
			</span>
		</a>
		<div class="locales" role="group" aria-label={m.localeLabel()}>
			<button type="button" aria-pressed={pressed('pt-BR')} onclick={() => chooseLocale('pt-BR')}>
				{m.localePt()}
			</button>
			<button type="button" aria-pressed={pressed('en')} onclick={() => chooseLocale('en')}>
				{m.localeEn()}
			</button>
		</div>
	</header>
	<main>
		{@render children()}
	</main>
</div>
