<script>
	import { onMount } from 'svelte';
	import Sparkline from '$lib/Sparkline.svelte';
	import TopList from '$lib/TopList.svelte';
	import {
		fetchTimeseries,
		fetchTop,
		fetchClients,
		fetchLatency
	} from '$lib/api.js';

	let status = null;
	let devices = [];
	let buckets = [];
	let topBlocked = [];
	let topAllowed = [];
	let topClients = [];
	let latency = null;
	let loading = true;
	let fetchError = false;

	let trusting = {};

	async function loadAll() {
		const [statusRes, devicesRes, ts, blocked, allowed, clients, lat] = await Promise.all([
			fetch('/api/status').then((r) => r.json()),
			fetch('/api/devices').then((r) => r.json()),
			fetchTimeseries('1h', '60s'),
			fetchTop('blocked', 10, '24h'),
			fetchTop('allowed', 10, '24h'),
			fetchClients(10, '24h'),
			fetchLatency('24h')
		]);
		status = statusRes;
		devices = devicesRes;
		buckets = ts.buckets;
		topBlocked = blocked.rows;
		topAllowed = allowed.rows;
		topClients = clients.rows;
		latency = lat.percentiles;
	}

	onMount(async () => {
		try {
			await loadAll();
		} catch {
			fetchError = true;
		} finally {
			loading = false;
		}

		// Refresh stats every 30s without a full reload.
		const refresh = setInterval(async () => {
			try {
				await loadAll();
			} catch {
				// transient errors are ignored on refresh
			}
		}, 30000);
		return () => clearInterval(refresh);
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
				devices = devices.map((d) => (d.mac === mac ? { ...d, trusted: true } : d));
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
	<a href="/" class="back">&larr; Tillbaka till startsidan</a>
	<header class="page-head">
		<h1>Avancerat läge</h1>
		<a href="/nerd" class="nerd-link" title="Live SSE feed, percentiler och Go-runtime">Nördläge →</a>
	</header>

	{#if loading}
		<p class="muted">Laddar data...</p>
	{:else if fetchError}
		<p class="error-msg" role="alert">Kunde inte hämta data från backend. Kontrollera att piholsterd körs.</p>
	{:else}
		<section class="card">
			<h2>Statistik (senaste 24 h)</h2>
			<div class="stat-grid">
				<div>
					<span class="stat-num">{status.blocked_today.toLocaleString('sv-SE')}</span>
					<span class="stat-label">blockerade</span>
				</div>
				<div>
					<span class="stat-num">{status.total_today.toLocaleString('sv-SE')}</span>
					<span class="stat-label">totalt</span>
				</div>
				{#if latency && latency.sample > 0}
					<div>
						<span class="stat-num">{latency.p50}<span class="unit">ms</span></span>
						<span class="stat-label">median latency (p50)</span>
					</div>
				{/if}
			</div>
			<div class="bar-wrap" role="img" aria-label="{blockedPct}% blockerat">
				<div class="bar-fill" style="width:{barWidth}"></div>
			</div>
			<p class="bar-label">{blockedPct}% av alla förfrågningar blockerades</p>
		</section>

		<section class="card">
			<h2>Senaste timmen</h2>
			<Sparkline {buckets} height={140} detailed={true} />
		</section>

		<div class="grid">
			<TopList rows={topBlocked} title="Topp blockerade domäner (24h)" monoLabel />
			<TopList rows={topAllowed} title="Topp tillåtna domäner (24h)" monoLabel />
			<TopList rows={topClients} title="Mest aktiva klienter (24h)" valueLabel="Per IP-adress" monoLabel />
		</div>

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
	}

	main {
		max-width: 900px;
		width: 100%;
		margin: 0 auto;
		padding: 2.5rem 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
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

	.page-head {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.75rem;
	}

	h1 {
		font-size: 2rem;
		margin: 0;
	}

	.nerd-link {
		color: #b08af7;
		font-size: 0.9rem;
		text-decoration: none;
		border: 1px solid #4a3a7a;
		border-radius: 0.4rem;
		padding: 0.4rem 0.75rem;
		transition: background 0.15s;
	}

	.nerd-link:hover {
		background: #2a1a4a;
	}

	h2 {
		font-size: 0.85rem;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: #aaa;
		margin: 0 0 1rem;
	}

	.card {
		background: #1a1a2e;
		border-radius: 0.75rem;
		padding: 1.5rem;
	}

	.stat-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
		gap: 1rem;
		margin-bottom: 1rem;
	}

	.stat-grid > div {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.stat-num {
		font-size: 1.8rem;
		font-weight: bold;
		color: #7eb8f7;
		font-variant-numeric: tabular-nums;
	}

	.unit {
		font-size: 0.95rem;
		color: #aaa;
		font-weight: normal;
		margin-left: 0.15rem;
	}

	.stat-label {
		color: #aaa;
		font-size: 0.85rem;
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

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: 1rem;
	}

	.device-list {
		list-style: none;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		padding: 0;
		margin: 0;
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
		color: #888;
	}

	.error-msg {
		color: #f5a0a0;
		background: #3a1010;
		padding: 0.75rem 1rem;
		border-radius: 0.5rem;
	}

	@media (max-width: 480px) {
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
