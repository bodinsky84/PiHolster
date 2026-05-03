import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		// BB-07: emit all styles as external files — enables strict 'self' CSP
		inlineStyleThreshold: 0,
		adapter: adapter({
			// Output goes into the api/dist directory so go:embed picks it up.
			pages: '../../apps/piholsterd/internal/api/dist',
			assets: '../../apps/piholsterd/internal/api/dist',
			fallback: 'index.html',
			precompress: false,
			strict: true
		}),
		// No SSR: piholsterd serves pre-built static files
		prerender: {
			handleHttpError: 'warn'
		}
	}
};

export default config;
