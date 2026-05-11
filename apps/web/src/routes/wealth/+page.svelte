<script>
	import { onMount, onDestroy } from 'svelte';

	let market = null;
	let signals = [];
	let loading = true;
	let error = null;

	async function fetchData() {
		try {
			const [mRes, sRes] = await Promise.all([
				fetch('/api/wealth/market'),
				fetch('/api/wealth/signals')
			]);

			if (!mRes.ok || !sRes.ok) {
				if (mRes.status === 401 || sRes.status === 401) {
					window.location.href = '/admin';
					return;
				}
				throw new Error('Failed to fetch wealth data');
			}

			market = await mRes.json();
			signals = await sRes.json();
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	let interval;
	onMount(() => {
		fetchData();
		interval = setInterval(fetchData, 30000); // Refresh every 30s
	});

	onDestroy(() => {
		if (interval) clearInterval(interval);
	});

	function formatTime(ts) {
		return new Date(ts).toLocaleTimeString();
	}
</script>

<svelte:head>
	<title>Wealth Terminal — PiHolster Alpha</title>
</svelte:head>

<div class="wealth-bg">
	<main class="terminal">
		<header class="header">
			<div class="brand">PIHOLSTER <span class="gold">ALPHA</span> v1.0</div>
			<nav>
				<a href="/advanced">EXIT_TERMINAL</a>
			</nav>
		</header>

		{#if loading && !market}
			<div class="loading-screen">
				<div class="spinner"></div>
				<p>INITIALIZING WEALTH_INTELLIGENCE_ENGINE...</p>
			</div>
		{:else if error}
			<div class="error-box">
				<h2>CRITICAL_ERROR</h2>
				<p>{error}</p>
				<button on:click={fetchData}>RETRY_SYNC</button>
			</div>
		{:else}
			<section class="market-ticker">
				<div class="ticker-item">
					<span class="label">BTC/USD</span>
					<span class="value">${market.bitcoin.toLocaleString()}</span>
				</div>
				<div class="ticker-item">
					<span class="label">ETH/USD</span>
					<span class="value">${market.ethereum.toLocaleString()}</span>
				</div>
				<div class="ticker-item">
					<span class="label">SOL/USD</span>
					<span class="value">${market.solana.toLocaleString()}</span>
				</div>
			</section>

			<div class="grid">
				<section class="signals-panel">
					<h3>ALPHA_SIGNALS</h3>
					<div class="signal-list">
						{#each signals as s}
							<div class="signal-card {s.type.toLowerCase()}">
								<div class="signal-header">
									<span class="type">{s.type}</span>
									<span class="prob">{Math.round(s.probability * 100)}% CONFIDENCE</span>
								</div>
								<p class="desc">{s.description}</p>
								<div class="time">{formatTime(s.timestamp)}</div>
							</div>
						{/each}
						{#if signals.length === 0}
							<p class="empty">Scanning market depth for opportunities...</p>
						{/if}
					</div>
				</section>

				<section class="analysis-panel">
					<div class="disclaimer">
						[NOTICE] WEALTH_INTELLIGENCE_ENGINE generates algorithmic estimates for informational purposes only. Trading involves high risk.
					</div>
					<h3>MARKET_ANALYSIS</h3>
					<div class="stats-grid">
						<div class="stat">
							<label>FEAR_GREED_INDEX</label>
							<div class="val gold">78 (Extreme Greed)</div>
						</div>
						<div class="stat">
							<label>24H_VOLUME</label>
							<div class="val">$142.4B</div>
						</div>
						<div class="stat">
							<label>DOMINANCE</label>
							<div class="val">BTC 54.2% / ETH 17.1%</div>
						</div>
					</div>

					<div class="chart-placeholder">
						<div class="grid-lines"></div>
						<div class="line"></div>
						<div class="label">REAL_TIME_MOMENTUM_OSCILLATOR</div>
					</div>
				</section>
			</div>
		{/if}
	</main>
</div>

<style>
	:global(body) {
		margin: 0;
		padding: 0;
		background: #050505;
		color: #e0e0e0;
		font-family: 'Inter', system-ui, -apple-system, sans-serif;
	}

	.wealth-bg {
		min-height: 100vh;
		background: radial-gradient(circle at top right, #1a1a00 0%, #050505 50%);
		padding: 1.5rem;
	}

	.terminal {
		max-width: 1200px;
		margin: 0 auto;
		background: rgba(10, 10, 10, 0.8);
		border: 1px solid #333;
		border-radius: 4px;
		box-shadow: 0 0 40px rgba(0, 0, 0, 0.5);
		display: flex;
		flex-direction: column;
		min-height: 85vh;
	}

	.header {
		padding: 1rem 1.5rem;
		border-bottom: 1px solid #333;
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.brand {
		font-weight: 900;
		letter-spacing: 2px;
		font-size: 1.2rem;
	}

	.gold {
		color: #ffd700;
		text-shadow: 0 0 10px rgba(255, 215, 0, 0.5);
	}

	nav a {
		color: #666;
		text-decoration: none;
		font-size: 0.8rem;
		font-family: monospace;
		border: 1px solid #333;
		padding: 4px 12px;
	}

	nav a:hover {
		color: #fff;
		border-color: #666;
	}

	.market-ticker {
		display: flex;
		background: #111;
		padding: 0.75rem 1.5rem;
		gap: 2rem;
		border-bottom: 1px solid #222;
	}

	.ticker-item {
		display: flex;
		gap: 0.5rem;
		align-items: baseline;
	}

	.ticker-item .label {
		font-size: 0.7rem;
		color: #666;
		font-weight: bold;
	}

	.ticker-item .value {
		font-family: monospace;
		font-weight: bold;
		color: #ffd700;
	}

	.grid {
		display: grid;
		grid-template-columns: 450px 1fr;
		flex: 1;
	}

	h3 {
		font-size: 0.8rem;
		color: #666;
		padding: 1rem 1.5rem;
		margin: 0;
		border-bottom: 1px solid #222;
		letter-spacing: 1px;
	}

	.signals-panel {
		border-right: 1px solid #333;
		display: flex;
		flex-direction: column;
	}

	.signal-list {
		padding: 1rem;
		overflow-y: auto;
		flex: 1;
	}

	.signal-card {
		background: #151515;
		border-left: 3px solid #444;
		padding: 1rem;
		margin-bottom: 1rem;
		border-radius: 2px;
	}

	.signal-card.arbitrage { border-left-color: #ffd700; }
	.signal-card.whale { border-left-color: #0af; }
	.signal-card.momentum { border-left-color: #0f0; }

	.signal-header {
		display: flex;
		justify-content: space-between;
		font-size: 0.7rem;
		font-weight: bold;
		margin-bottom: 0.5rem;
	}

	.type { color: #ffd700; }
	.prob { color: #666; }

	.desc {
		font-size: 0.9rem;
		margin: 0.5rem 0;
		line-height: 1.4;
	}

	.time {
		font-size: 0.7rem;
		color: #444;
		text-align: right;
	}

	.analysis-panel {
		padding: 0;
	}

	.disclaimer {
		background: #2a1a00;
		color: #ffd700;
		font-size: 0.7rem;
		padding: 0.5rem 1rem;
		border-bottom: 1px solid #333;
		font-weight: bold;
	}

	.stats-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		padding: 1.5rem;
		gap: 1.5rem;
	}

	.stat label {
		display: block;
		font-size: 0.65rem;
		color: #444;
		margin-bottom: 0.5rem;
		font-weight: bold;
	}

	.stat .val {
		font-size: 1.1rem;
		font-weight: bold;
	}

	.chart-placeholder {
		margin: 0 1.5rem 1.5rem;
		height: 300px;
		background: #0a0a0a;
		border: 1px solid #222;
		position: relative;
		overflow: hidden;
	}

	.grid-lines {
		position: absolute;
		inset: 0;
		background-image: linear-gradient(#111 1px, transparent 1px), linear-gradient(90deg, #111 1px, transparent 1px);
		background-size: 20px 20px;
	}

	.chart-placeholder .line {
		position: absolute;
		bottom: 100px;
		left: 0;
		width: 100%;
		height: 2px;
		background: linear-gradient(90deg, transparent, #ffd700, transparent);
		box-shadow: 0 0 15px #ffd700;
	}

	.chart-placeholder .label {
		position: absolute;
		top: 10px;
		right: 10px;
		font-size: 0.6rem;
		color: #333;
		font-weight: bold;
	}

	.loading-screen {
		flex: 1;
		display: flex;
		flex-direction: column;
		justify-content: center;
		align-items: center;
		gap: 1rem;
		color: #ffd700;
	}

	.spinner {
		width: 40px;
		height: 40px;
		border: 2px solid #333;
		border-top-color: #ffd700;
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	@keyframes spin { to { transform: rotate(360deg); } }

	@media (max-width: 900px) {
		.grid { grid-template-columns: 1fr; }
		.signals-panel { border-right: none; border-bottom: 1px solid #333; }
	}
</style>
