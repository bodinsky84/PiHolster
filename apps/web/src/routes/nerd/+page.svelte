<script>
	import { onMount, onDestroy } from 'svelte';
	import Sparkline from '$lib/Sparkline.svelte';
	import TopList from '$lib/TopList.svelte';
	import {
		fetchTimeseries,
		fetchTop,
		fetchClients,
		fetchLatency,
		fetchSystem,
		subscribeLive
	} from '$lib/api.js';

	let buckets = [];
	let topBlocked = [];
	let topClients = [];
	let latency = null;
	let system = null;

	let liveEvents = [];
	let liveMax = 200;
	let paused = false;
	let filter = '';
	let regexError = '';

	let showHelp = false;
	let loading = true;
	let loadError = false;

	let unsub = null;
	let refreshTimer = null;
	let systemTimer = null;

	async function loadStats() {
		const [ts, blocked, clients, lat] = await Promise.all([
			fetchTimeseries('24h', '300s'),
			fetchTop('blocked', 15, '24h'),
			fetchClients(15, '24h'),
			fetchLatency('24h')
		]);
		buckets = ts.buckets;
		topBlocked = blocked.rows;
		topClients = clients.rows;
		latency = lat.percentiles;
	}

	async function loadSystem() {
		try {
			system = await fetchSystem();
		} catch {
			// keep last known value
		}
	}

	function handleLive(e) {
		if (paused) return;
		liveEvents = [e, ...liveEvents].slice(0, liveMax);
	}

	$: filterRegex = (() => {
		regexError = '';
		if (!filter) return null;
		try {
			return new RegExp(filter, 'i');
		} catch (err) {
			regexError = String(err.message ?? err);
			return null;
		}
	})();

	$: filteredEvents = filterRegex
		? liveEvents.filter((e) => filterRegex.test(e.domain) || filterRegex.test(e.client_ip))
		: liveEvents;

	$: blockedRate = (() => {
		if (liveEvents.length === 0) return 0;
		const blocked = liveEvents.filter((e) => e.blocked).length;
		return Math.round((blocked / liveEvents.length) * 100);
	})();

	function formatUptime(s) {
		if (!s) return '0s';
		const d = Math.floor(s / 86400);
		const h = Math.floor((s % 86400) / 3600);
		const m = Math.floor((s % 3600) / 60);
		const sec = s % 60;
		if (d > 0) return `${d}d ${h}h`;
		if (h > 0) return `${h}h ${m}m`;
		if (m > 0) return `${m}m ${sec}s`;
		return `${sec}s`;
	}

	function formatTime(ts) {
		const d = new Date(ts);
		return d.toLocaleTimeString('sv-SE', { hour12: false }) + '.' + String(d.getMilliseconds()).padStart(3, '0');
	}

	function onKeydown(e) {
		// Don't hijack typing in the filter box.
		if (e.target instanceof HTMLInputElement) {
			if (e.key === 'Escape') e.target.blur();
			return;
		}
		switch (e.key) {
			case 'p':
				paused = !paused;
				break;
			case 'c':
				liveEvents = [];
				break;
			case '/':
				e.preventDefault();
				document.getElementById('filter-input')?.focus();
				break;
			case '?':
				showHelp = !showHelp;
				break;
			case 'Escape':
				showHelp = false;
				break;
		}
	}

	onMount(async () => {
		try {
			await Promise.all([loadStats(), loadSystem()]);
		} catch {
			loadError = true;
		} finally {
			loading = false;
		}

		unsub = subscribeLive(handleLive, { replay: 50 });
		refreshTimer = setInterval(() => loadStats().catch(() => {}), 30000);
		systemTimer = setInterval(loadSystem, 5000);

		window.addEventListener('keydown', onKeydown);
	});

	onDestroy(() => {
		unsub?.();
		if (refreshTimer) clearInterval(refreshTimer);
		if (systemTimer) clearInterval(systemTimer);
		if (typeof window !== 'undefined') window.removeEventListener('keydown', onKeydown);
	});
</script>

<svelte:head>
	<title>Nördläge — PiHolster</title>
</svelte:head>

<main>
	<header class="page-head">
		<a href="/advanced" class="back">&larr; Avancerat</a>
		<h1>Nörd<span class="accent">läge</span></h1>
		<button class="help-btn" on:click={() => (showHelp = !showHelp)} title="Hjälp (?)">?</button>
	</header>

	{#if loading}
		<p class="muted">Laddar...</p>
	{:else if loadError}
		<p class="error-msg">Kunde inte hämta data. Kontrollera att piholsterd körs.</p>
	{:else}
		<section class="card chart-card">
			<header class="card-head">
				<h2>Frågor senaste 24h <span class="muted">(5-min buckets)</span></h2>
				{#if latency && latency.sample > 0}
					<div class="latency-pills">
						<span class="pill">p50: <b>{latency.p50}ms</b></span>
						<span class="pill">p95: <b>{latency.p95}ms</b></span>
						<span class="pill">p99: <b>{latency.p99}ms</b></span>
						<span class="pill">max: <b>{latency.max}ms</b></span>
					</div>
				{/if}
			</header>
			<Sparkline {buckets} height={200} detailed={true} />
		</section>

		<section class="card live-card">
			<header class="card-head live-head">
				<h2>Live-flöde</h2>
				<div class="live-controls">
					<input
						id="filter-input"
						bind:value={filter}
						placeholder="Regex-filter (tryck / för fokus)"
						class="filter-input"
						class:invalid={regexError}
					/>
					<button class="ctl" on:click={() => (paused = !paused)} title="P för paus">
						{paused ? '▶ Återuppta' : '⏸ Pausa'}
					</button>
					<button class="ctl" on:click={() => (liveEvents = [])} title="C för rensa">Rensa</button>
				</div>
			</header>

			{#if regexError}
				<p class="error-inline">Ogiltigt regex: {regexError}</p>
			{/if}

			<div class="live-stats">
				<span><b>{filteredEvents.length}</b> / {liveEvents.length} händelser</span>
				<span class="sep">•</span>
				<span>{blockedRate}% blockerade i bufferten</span>
				{#if paused}
					<span class="sep">•</span>
					<span class="paused-tag">PAUSAD</span>
				{/if}
			</div>

			<ol class="feed">
				{#each filteredEvents as e, i (e.ts + ':' + i)}
					<li class="event" class:blocked={e.blocked}>
						<span class="time">{formatTime(e.ts)}</span>
						<span class="dot" class:blocked={e.blocked} class:allowed={!e.blocked}></span>
						<span class="domain" title={e.domain}>{e.domain}</span>
						<span class="ip">{e.client_ip}</span>
						{#if !e.blocked}
							<span class="latency">{e.latency_ms}ms</span>
						{:else}
							<span class="latency blocked-tag">NXDOMAIN</span>
						{/if}
					</li>
				{:else}
					<li class="empty muted">Väntar på DNS-frågor... (replayar de senaste 50 vid uppkoppling)</li>
				{/each}
			</ol>
		</section>

		<div class="grid">
			<TopList rows={topBlocked} title="Topp blockerade (24h)" monoLabel />
			<TopList rows={topClients} title="Mest aktiva klienter (24h)" monoLabel />

			<section class="card system-card">
				<h2>Runtime</h2>
				{#if system}
					<dl class="kv">
						<dt>Uptime</dt>
						<dd>{formatUptime(system.uptime_seconds)}</dd>
						<dt>Go</dt>
						<dd>{system.go_version}</dd>
						<dt>Goroutines</dt>
						<dd>{system.goroutines}</dd>
						<dt>Heap</dt>
						<dd>{system.heap_alloc_mb} MB</dd>
						<dt>Sys</dt>
						<dd>{system.sys_mb} MB</dd>
						<dt>GC</dt>
						<dd>{system.num_gc.toLocaleString('sv-SE')}</dd>
					</dl>
				{:else}
					<p class="muted">Ingen data.</p>
				{/if}
			</section>
		</div>
	{/if}
</main>

{#if showHelp}
	<div
		class="modal-backdrop"
		on:click={() => (showHelp = false)}
		on:keydown={(e) => e.key === 'Escape' && (showHelp = false)}
		role="button"
		tabindex="-1"
	>
		<div class="modal" on:click|stopPropagation role="dialog" aria-modal="true">
			<h3>Tangentbord</h3>
			<dl class="shortcuts">
				<dt>p</dt><dd>Pausa / återuppta live-flödet</dd>
				<dt>c</dt><dd>Rensa bufferten</dd>
				<dt>/</dt><dd>Fokusera filterfältet</dd>
				<dt>?</dt><dd>Visa/dölj denna ruta</dd>
				<dt>Esc</dt><dd>Stäng / lämna fältet</dd>
			</dl>
			<p class="muted">Filtret är ett JavaScript-regex som matchar mot domän eller klient-IP.</p>
			<button class="ctl" on:click={() => (showHelp = false)}>Stäng</button>
		</div>
	</div>
{/if}

<style>
	:global(body) {
		font-family: system-ui, sans-serif;
		background: #0a0a14;
		color: #e0e0e0;
		min-height: 100vh;
	}

	main {
		max-width: 1100px;
		width: 100%;
		margin: 0 auto;
		padding: 2rem 1.5rem 4rem;
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.page-head {
		display: flex;
		align-items: center;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.back {
		color: #7eb8f7;
		text-decoration: none;
	}

	h1 {
		font-size: 2rem;
		margin: 0;
		flex: 1;
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		letter-spacing: -0.02em;
	}

	.accent {
		color: #b08af7;
	}

	.help-btn {
		width: 32px;
		height: 32px;
		border-radius: 50%;
		background: #2a1a4a;
		color: #b08af7;
		border: 1px solid #4a3a7a;
		font-weight: bold;
		cursor: pointer;
	}

	h2 {
		font-size: 0.82rem;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: #aaa;
		margin: 0;
		font-weight: 600;
	}

	.card {
		background: #14142a;
		border: 1px solid #1f1f3a;
		border-radius: 0.6rem;
		padding: 1.25rem 1.5rem;
	}

	.card-head {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
		flex-wrap: wrap;
		margin-bottom: 0.75rem;
	}

	.muted {
		color: #777;
		font-size: 0.85rem;
		font-weight: normal;
		text-transform: none;
		letter-spacing: 0;
	}

	.latency-pills {
		display: flex;
		gap: 0.4rem;
		flex-wrap: wrap;
	}

	.pill {
		font-size: 0.78rem;
		background: #1a1a35;
		padding: 0.2rem 0.55rem;
		border-radius: 999px;
		color: #aaa;
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
	}

	.pill b {
		color: #7eb8f7;
	}

	.live-head {
		flex-wrap: wrap;
	}

	.live-controls {
		display: flex;
		gap: 0.5rem;
		align-items: center;
		flex-wrap: wrap;
	}

	.filter-input {
		background: #0a0a14;
		border: 1px solid #2a2a4a;
		color: #e0e0e0;
		padding: 0.4rem 0.75rem;
		border-radius: 0.4rem;
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		font-size: 0.85rem;
		min-width: 220px;
	}

	.filter-input.invalid {
		border-color: #e74c3c;
	}

	.ctl {
		background: #1f1f3a;
		color: #b0c4f7;
		border: 1px solid #2a2a4a;
		border-radius: 0.4rem;
		padding: 0.4rem 0.75rem;
		font-size: 0.82rem;
		cursor: pointer;
	}

	.ctl:hover {
		background: #2a2a4a;
	}

	.error-inline {
		color: #f5a0a0;
		font-size: 0.85rem;
		margin: 0 0 0.5rem;
	}

	.live-stats {
		display: flex;
		gap: 0.5rem;
		font-size: 0.82rem;
		color: #888;
		margin-bottom: 0.5rem;
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
	}

	.sep {
		color: #444;
	}

	.paused-tag {
		color: #f0b75c;
		font-weight: bold;
	}

	.feed {
		list-style: none;
		padding: 0;
		margin: 0;
		max-height: 480px;
		overflow-y: auto;
		background: #0a0a14;
		border-radius: 0.4rem;
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		font-size: 0.82rem;
	}

	.event {
		display: grid;
		grid-template-columns: 90px 14px 1fr 130px 70px;
		gap: 0.5rem;
		padding: 0.3rem 0.75rem;
		border-bottom: 1px solid #14142a;
		align-items: center;
	}

	.event.blocked {
		background: rgba(231, 76, 60, 0.06);
	}

	.time {
		color: #555;
	}

	.dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
	}

	.dot.blocked {
		background: #e74c3c;
	}

	.dot.allowed {
		background: #52b788;
	}

	.domain {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		color: #d0d0d0;
	}

	.ip {
		color: #888;
		text-align: right;
	}

	.latency {
		text-align: right;
		color: #7eb8f7;
	}

	.latency.blocked-tag {
		color: #e74c3c;
	}

	.empty {
		padding: 1rem;
		text-align: center;
	}

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: 1rem;
	}

	.system-card .kv {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 0.4rem 1rem;
		margin: 0;
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		font-size: 0.85rem;
	}

	.system-card dt {
		color: #888;
	}

	.system-card dd {
		margin: 0;
		color: #d0d0d0;
		text-align: right;
	}

	.error-msg {
		color: #f5a0a0;
		background: #3a1010;
		padding: 0.75rem 1rem;
		border-radius: 0.5rem;
	}

	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.7);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 100;
	}

	.modal {
		background: #14142a;
		border: 1px solid #2a2a4a;
		border-radius: 0.6rem;
		padding: 1.5rem 2rem;
		max-width: 420px;
		width: calc(100% - 2rem);
	}

	.modal h3 {
		margin: 0 0 1rem;
		color: #b08af7;
	}

	.shortcuts {
		display: grid;
		grid-template-columns: 60px 1fr;
		gap: 0.5rem 1rem;
		margin: 0 0 1rem;
		font-size: 0.9rem;
	}

	.shortcuts dt {
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		background: #1f1f3a;
		padding: 0.15rem 0.5rem;
		border-radius: 0.3rem;
		text-align: center;
		color: #b08af7;
		font-weight: bold;
	}

	.shortcuts dd {
		margin: 0;
		color: #d0d0d0;
		align-self: center;
	}

	@media (max-width: 640px) {
		.event {
			grid-template-columns: 80px 10px 1fr 60px;
		}
		.event .ip {
			display: none;
		}
		.filter-input {
			min-width: 0;
			flex: 1;
		}
	}
</style>
