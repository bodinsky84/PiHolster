<script>
	import { onMount } from 'svelte';
	let lines = ["PIHOLSTER OS v0.1.0-NERD INITIALIZING...", "BOOTING PASSIVE INCOME MODULE...", "SYSTEM: OPTIMAL"];
</script>
<div style="background: #000; color: #0f0; font-family: monospace; min-height: 100vh; padding: 2rem;">
	{#each lines as line}
		<div>{line}</div>
	{/each}
	<div style="margin-top: 2rem;">
		<a href="/advanced" style="color: #0f0;">[EXIT]</a>
	</div>
</div>
