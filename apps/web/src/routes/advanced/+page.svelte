<script>
	import { onMount } from 'svelte';

	let status = null;
	let devices = [];
	let loading = true;
	let fetchError = false;

	// Tracks which trust-POSTs are in flight per MAC to prevent double-clicks.
	let trusting = {};

	onMount(async () => {
		try {
			const [statusRes, devicesRes] = await Promise.all([
				fetch('/api/status'),
				fetch('/api/devices')
			]);
			if (!statusRes.ok || !devicesRes.ok) throw new Error('http error');
			status = await statusRes.json();
			devices = await devicesRes.json();
		} catch {
			fetchError = true;
		} finally {
			loading = false;
		}
	});

	async function trustDevice(mac) {
		trusting = { ...trusting, [mac]: true };
		try {
			const res = await fetch(`/api/devices/${encodeURIComponent(mac)}/trust`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ trusted: true })
			});
			if (res.ok || res.status === 204) {
				devices = devices.map((d) =>
					d.mac === mac ? { ...d, trusted: true } : d
				);
			}
		} finally {
			trusting = { ...trusting, [mac]: false };
		}
	}

	function deviceIcon(hostname) {
		if (!hostname) return '❓';
		const h = hostname.toLowerCase();
		if (h.includes('phone') || h.includes('iphone') || h.includes('android')) return '📱';
		if (h.includes('tv') || h.includes('roku') || h.includes('fire') || h.includes('chromecast')) return '📺';
		if (h.includes('mac') || h.includes('pc') || h.includes('laptop') || h.includes('desktop')) return '💻';
		return '❓';
	}

	function displayName(d) {
		return d.display_name || d.hostname || d.mac;
	}

	$: blockedPct =
		status && status.total_today > 0
			? Math.round((status.blocked_today / status.total_today) * 100)
			: 0;

	$: barWidth = blockedPct + '%';
</script>

<svelte:head>
	<title>Avancerat — PiHolster</title>
</svelte:head>

<main>
	<div class="nav-header">
		<a href="/" class="back">&larr; Tillbaka till startsidan</a>
		<div class="links">
			<a href="/wealth" class="wealth-link">WEALTH_TERMINAL</a>
			<a href="/nerd" class="nerd-link">Nördläge (AAAAA)</a>
		</div>
	</div>
	<h1>Avancerat läge</h1>

	{#if loading}
		<p class="muted">Laddar data...</p>
	{:else if fetchError}
		<p class="error-msg" role="alert">Kunde inte hämta data från backend. Kontrollera att piholsterd körs.</p>
	{:else}
		<section class="card">
			<h2>Statistik (senaste 24 h)</h2>
			<div class="stat-row">
				<span class="stat-num">{status.blocked_today.toLocaleString('sv-SE')}</span>
				<span class="stat-label">blockerade förfrågningar</span>
			</div>
			<div class="stat-row">
				<span class="stat-num">{status.total_today.toLocaleString('sv-SE')}</span>
				<span class="stat-label">totalt</span>
			</div>
			<div class="bar-wrap" role="img" aria-label="{blockedPct}% blockerat">
				<div class="bar-fill" style="width:{barWidth}"></div>
			</div>
			<p class="bar-label">{blockedPct}% av alla förfrågningar blockerades</p>
		</section>

		<section class="card">
			<h2>Enheter ({devices.length})</h2>
			{#if devices.length === 0}
				<p class="muted">Inga enheter har setts ännu.</p>
			{:else}
				<ul class="device-list">
					{#each devices as d (d.mac)}
						<li class="device-item">
							<span class="device-icon" aria-hidden="true">{deviceIcon(d.hostname)}</span>
							<div class="device-info">
								<span class="device-name">{displayName(d)}</span>
								<span class="device-ip">{d.ip}</span>
							</div>
							<div class="device-right">
								{#if d.trusted}
									<span class="badge badge-trusted">Betrodd</span>
								{:else}
									<span class="badge badge-unknown">Okänd</span>
									<button
										class="btn-trust"
										disabled={trusting[d.mac]}
										on:click={() => trustDevice(d.mac)}
									>
										{trusting[d.mac] ? 'Sparar...' : 'Det är okej — min enhet'}
									</button>
								{/if}
							</div>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}
</main>

<style>
	:global(body) {
		font-family: system-ui, sans-serif;
		background: #0f0f1a;
		color: #e0e0e0;
		min-height: 100vh;
		display: flex;
		justify-content: center;
		align-items: flex-start;
	}

	main {
		max-width: 700px;
		width: 100%;
		padding: 2.5rem 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 2rem;
	}

	.back {
		display: inline-block;
		color: #7eb8f7;
		text-decoration: none;
		font-size: 1rem;
	}

	.back:hover {
		text-decoration: underline;
	}

	.nav-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.links {
		display: flex;
		gap: 1rem;
	}

	.wealth-link {
		color: #ffd700;
		text-decoration: none;
		font-family: monospace;
		font-weight: bold;
		border: 1px solid #ffd700;
		padding: 0.25rem 0.5rem;
		border-radius: 4px;
		font-size: 0.8rem;
		transition: all 0.2s;
	}

	.wealth-link:hover {
		background: #ffd700;
		color: #000;
		box-shadow: 0 0 10px #ffd700;
	}

	.nerd-link {
		color: #0f0;
		text-decoration: none;
		font-family: monospace;
		font-weight: bold;
		border: 1px solid #0f0;
		padding: 0.25rem 0.5rem;
		border-radius: 4px;
		font-size: 0.8rem;
		transition: all 0.2s;
	}

	.nerd-link:hover {
		background: #0f0;
		color: #000;
		box-shadow: 0 0 10px #0f0;
	}

	h1 {
		font-size: 2rem;
	}

	h2 {
		font-size: 1.2rem;
		margin-bottom: 1rem;
		color: #aaa;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		font-size: 0.85rem;
	}

	.card {
		background: #1a1a2e;
		border-radius: 0.75rem;
		padding: 1.5rem;
	}

	.stat-row {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
		margin-bottom: 0.4rem;
	}

	.stat-num {
		font-size: 1.8rem;
		font-weight: bold;
		color: #7eb8f7;
	}

	.stat-label {
		color: #aaa;
		font-size: 0.95rem;
	}

	.bar-wrap {
		height: 14px;
		background: #2a2a3e;
		border-radius: 7px;
		overflow: hidden;
		margin: 1rem 0 0.4rem;
	}

	.bar-fill {
		height: 100%;
		background: #e74c3c;
		border-radius: 7px;
		transition: width 0.5s ease;
	}

	.bar-label {
		font-size: 0.85rem;
		color: #aaa;
	}

	.device-list {
		list-style: none;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.device-item {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		background: #0f0f1a;
		border-radius: 0.5rem;
		padding: 0.75rem 1rem;
		flex-wrap: wrap;
	}

	.device-icon {
		font-size: 1.5rem;
		flex-shrink: 0;
	}

	.device-info {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
		flex: 1;
		min-width: 0;
	}

	.device-name {
		font-weight: bold;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.device-ip {
		font-size: 0.8rem;
		color: #888;
		font-family: monospace;
	}

	.device-right {
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		gap: 0.4rem;
	}

	.badge {
		display: inline-block;
		padding: 0.2rem 0.6rem;
		border-radius: 0.3rem;
		font-size: 0.78rem;
		font-weight: bold;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.badge-trusted {
		background: #1b4332;
		color: #52b788;
	}

	.badge-unknown {
		background: #5a1a1a;
		color: #f5a0a0;
	}

	.btn-trust {
		background: #2c2c50;
		color: #b0c4f7;
		border: 1px solid #4a4a7a;
		border-radius: 0.4rem;
		padding: 0.4rem 0.75rem;
		font-size: 0.82rem;
		cursor: pointer;
		transition: background 0.15s;
		white-space: nowrap;
	}

	.btn-trust:hover:not(:disabled) {
		background: #3a3a6a;
	}

	.btn-trust:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.muted {
		color: #666;
	}

	.error-msg {
		color: #f5a0a0;
		background: #3a1010;
		padding: 0.75rem 1rem;
		border-radius: 0.5rem;
	}

	@media (max-width: 375px) {
		main {
			padding: 1.5rem 1rem;
		}

		.device-item {
			flex-direction: column;
			align-items: flex-start;
		}

		.device-right {
			align-items: flex-start;
		}

		.stat-num {
			font-size: 1.4rem;
		}
	}
</style>
