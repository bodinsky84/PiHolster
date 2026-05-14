<script>
	import { onMount } from 'svelte';

	let status = 'ok';
	let blockedToday = 0;
	let totalToday = 0;
	let devicesOnline = 0;
	let loading = true;
	let fetchError = false;

	onMount(async () => {
		try {
			const res = await fetch('/api/status');
			if (!res.ok) throw new Error('http ' + res.status);
			const data = await res.json();
			status = data.status ?? 'error';
			blockedToday = data.blocked_today ?? 0;
			totalToday = data.total_today ?? 0;
			devicesOnline = data.devices_online ?? 0;
		} catch {
			fetchError = true;
			status = 'error';
		} finally {
			loading = false;
		}
	});

	$: percentBlocked =
		totalToday > 0 ? Math.round((blockedToday / totalToday) * 100) : 0;

	$: circleColor =
		status === 'ok' ? '#2d6a4f' : status === 'warning' ? '#b5680a' : '#8b1a1a';

	$: ringInner =
		status === 'ok' ? '#b7e4c7' : status === 'warning' ? '#ffe5a0' : '#f5b8b8';

	$: ringOuter =
		status === 'ok' ? '#d8f3dc' : status === 'warning' ? '#fff3cd' : '#fde0e0';

	$: circleLabel =
		status === 'ok'
			? 'Ditt nätverk är skyddat'
			: status === 'warning'
			? 'Kontrollera ditt nätverk'
			: 'Problem hittades';
</script>

<svelte:head>
	<title>PiHolster</title>
</svelte:head>

<main>
	{#if loading}
		<div class="circle loading-circle" aria-busy="true" aria-label="Laddar nätverksstatus">
			<span class="circle-text">Laddar...</span>
		</div>
	{:else}
		<div
			class="circle"
			style="background:{circleColor}; box-shadow: 0 0 0 12px {ringInner}, 0 0 0 24px {ringOuter};"
			aria-label="Nätverksstatus: {circleLabel}"
			role="status"
		>
			<span class="circle-text">{circleLabel}</span>
		</div>
	{/if}

	{#if !loading && !fetchError}
		<div class="stats" aria-label="Statistik för idag">
			<div class="stat">
				<span class="stat-value">{blockedToday.toLocaleString('sv-SE')}</span>
				<span class="stat-label">Blockerade idag</span>
			</div>
			<div class="stat">
				<span class="stat-value">{devicesOnline}</span>
				<span class="stat-label">Enheter online</span>
			</div>
			<div class="stat">
				<span class="stat-value">{percentBlocked}%</span>
				<span class="stat-label">Andel blockerat</span>
			</div>
		</div>
	{/if}

	{#if fetchError}
		<p class="fetch-error" role="alert">
			Kunde inte nå backend. Kontrollera att piholsterd körs.
		</p>
	{/if}

	<nav>
		<a href="/allsvenskan" class="btn btn-allsvenskan">Allsvenskan &rarr;</a>
		<a href="/advanced" class="btn btn-advanced">Avancerat &rarr;</a>
		<a href="/admin" class="btn btn-admin">Admin &rarr;</a>
	</nav>
</main>

<style>
	:global(body) {
		background: #f0f4f0;
		font-family: Georgia, 'Times New Roman', serif;
		color: #111;
		min-height: 100vh;
		display: flex;
		justify-content: center;
		align-items: center;
	}

	main {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 2.5rem;
		padding: 2rem;
		width: 100%;
		max-width: 480px;
	}

	.circle {
		width: 260px;
		height: 260px;
		border-radius: 50%;
		display: flex;
		justify-content: center;
		align-items: center;
		transition: background 0.4s, box-shadow 0.4s;
	}

	.loading-circle {
		background: #aaa;
		box-shadow: 0 0 0 12px #ddd, 0 0 0 24px #eee;
	}

	.circle-text {
		color: #fff;
		font-size: 1.35rem;
		font-weight: bold;
		text-align: center;
		line-height: 1.4;
		padding: 1rem;
	}

	.stats {
		display: flex;
		gap: 1.5rem;
		justify-content: center;
		flex-wrap: wrap;
		width: 100%;
	}

	.stat {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.25rem;
		min-width: 90px;
	}

	.stat-value {
		font-size: 2rem;
		font-weight: bold;
		color: #2d6a4f;
		line-height: 1;
	}

	.stat-label {
		font-size: 0.85rem;
		color: #555;
		text-align: center;
	}

	.fetch-error {
		color: #8b1a1a;
		font-size: 1rem;
		text-align: center;
		background: #fde0e0;
		padding: 0.75rem 1rem;
		border-radius: 0.5rem;
		width: 100%;
	}

	nav {
		display: flex;
		flex-direction: column;
		gap: 1rem;
		width: 100%;
	}

	.btn {
		display: block;
		width: 100%;
		padding: 1.25rem 1.5rem;
		font-size: 1.4rem;
		font-family: inherit;
		font-weight: bold;
		border: none;
		border-radius: 0.75rem;
		text-align: center;
		text-decoration: none;
		cursor: pointer;
		transition: filter 0.15s;
	}

	.btn:hover {
		filter: brightness(0.9);
	}

	.btn-allsvenskan {
		background: #2d6a4f;
		color: #fff;
	}

	.btn-advanced {
		background: #1a1a2e;
		color: #fff;
	}

	.btn-admin {
		background: #c0392b;
		color: #fff;
	}

	@media (max-width: 375px) {
		.circle {
			width: 210px;
			height: 210px;
		}

		.circle-text {
			font-size: 1.1rem;
		}

		.stat-value {
			font-size: 1.6rem;
		}

		.btn {
			font-size: 1.2rem;
			padding: 1rem;
		}
	}
</style>
