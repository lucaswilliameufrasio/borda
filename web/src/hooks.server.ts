import type { Handle } from '@sveltejs/kit/hooks';
import { paraglideMiddleware } from '#lib/paraglide/server.js';
import { getTextDirection } from '#lib/paraglide/runtime.js';

const paraglideHandle: Handle = ({ event, resolve }) => {
	return paraglideMiddleware(event.request, ({ request: localizedRequest, locale }) => {
		return resolve(
			{ ...event, request: localizedRequest },
			{
				transformPageChunk: ({ html }) => {
					return html.replace('%lang%', locale).replace('%dir%', getTextDirection(locale));
				},
			},
		);
	});
};

export const handle: Handle = paraglideHandle;
