import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		port: 5173,
		proxy: {
			// Forward API calls to the Go backend during development
			'/api': {
				target: 'http://localhost:8080',
				changeOrigin: true
			}
		}
	},
	build: {
		target: 'es2020',
		cssCodeSplit: true
	}
});
