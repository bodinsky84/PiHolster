<script>
	// Global layout shell — per-page styles handle their own look and feel.
</script>

<slot />

<style>
	:global(*, *::before, *::after) {
		box-sizing: border-box;
		margin: 0;
		padding: 0;
	}

	:global(html, body) {
		height: 100%;
	}
</style>
