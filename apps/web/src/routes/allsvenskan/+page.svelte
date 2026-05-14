<script>
	import { onMount } from 'svelte';

	let table = [];
	let news = [];
	let matches = [];
	let stats = { top_scorers: [], top_cards: [] };
	let loading = true;
	let error = '';

	onMount(async () => {
		try {
			const [tableRes, newsRes, matchesRes, statsRes] = await Promise.all([
				fetch('/api/allsvenskan/table'),
				fetch('/api/allsvenskan/news'),
				fetch('/api/allsvenskan/matches'),
				fetch('/api/allsvenskan/stats')
			]);

			if (!tableRes.ok || !newsRes.ok || !matchesRes.ok || !statsRes.ok) {
				throw new Error('Kunde inte hämta data');
			}

			table = await tableRes.json();
			news = await newsRes.json();
			matches = await matchesRes.json();
			stats = await statsRes.json();
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	});

	function formatDate(dateStr) {
		const d = new Date(dateStr);
		return d.toLocaleDateString('sv-SE', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' });
	}
</script>

<svelte:head>
	<title>Allsvenskan — PiHolster</title>
</svelte:head>

<main>
	<a href="/" class="back">&larr; Tillbaka</a>

	<h1>Allsvenskan</h1>

	{#if loading}
		<p>Laddar statistik...</p>
	{:else if error}
		<p class="error">{error}</p>
	{:else}
		<section class="grid">
			<div class="card table-card">
				<h2>Tabell</h2>
				<div class="table-wrapper">
					<table>
						<thead>
							<tr>
								<th>#</th>
								<th>Lag</th>
								<th>S</th>
								<th>V</th>
								<th>O</th>
								<th>F</th>
								<th>Mål</th>
								<th>P</th>
							</tr>
						</thead>
						<tbody>
							{#each table as entry}
								<tr>
									<td>{entry.position}</td>
									<td class="team-name">{entry.team}</td>
									<td>{entry.games}</td>
									<td>{entry.wins}</td>
									<td>{entry.draws}</td>
									<td>{entry.losses}</td>
									<td>{entry.goals}</td>
									<td class="points">{entry.points}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>

			<div class="card matches-card">
				<h2>Senaste matcher</h2>
				<ul class="match-list">
					{#each (matches || []) as m}
						<li>
							<span class="match-date">{m.date}</span>
							<div class="match-teams">
								<span>{m.home_team}</span>
								<span class="match-result">{m.result}</span>
								<span>{m.away_team}</span>
							</div>
						</li>
					{/each}
				</ul>
			</div>

			<div class="card news-card">
				<h2>Senaste nyheter</h2>
				<ul class="news-list">
					{#each news as item}
						<li>
							<a href={item.link} target="_blank" rel="noopener">
								<span class="news-date">{formatDate(item.pub_date)}</span>
								<span class="news-title">{item.title}</span>
							</a>
							<p>{item.description}</p>
						</li>
					{/each}
				</ul>
			</div>

			<div class="card stats-card">
				<h2>Skytteliga</h2>
				<ul class="stat-list">
					{#each (stats.top_scorers || []) as s}
						<li>
							<span class="player">{s.name} ({s.team})</span>
							<span class="value">{s.value} mål</span>
						</li>
					{/each}
				</ul>

				<h2 class="mt-4">Kortliga</h2>
				<ul class="stat-list">
					{#each (stats.top_cards || []) as s}
						<li>
							<span class="player">{s.name} ({s.team})</span>
							<span class="value">{s.value} kort</span>
						</li>
					{/each}
				</ul>
			</div>
		</section>
	{/if}
</main>

<style>
	:global(body) {
		background: #0f172a;
		color: #f8fafc;
		font-family: 'Inter', sans-serif;
	}

	main {
		max-width: 1200px;
		margin: 0 auto;
		padding: 2rem;
	}

	.back {
		color: #94a3b8;
		text-decoration: none;
		margin-bottom: 1rem;
		display: inline-block;
	}

	h1 {
		font-size: 2.5rem;
		margin-bottom: 2rem;
		color: #38bdf8;
	}

	h2 {
		font-size: 1.25rem;
		margin-bottom: 1rem;
		color: #94a3b8;
	}

	.grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 2rem;
	}

	@media (max-width: 1024px) {
		.grid {
			grid-template-columns: 1fr;
		}
	}

	.card {
		background: #1e293b;
		padding: 1.5rem;
		border-radius: 1rem;
		box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
	}

	.table-card {
		grid-row: span 2;
	}

	.table-wrapper {
		overflow-x: auto;
	}

	table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.9rem;
	}

	th, td {
		text-align: left;
		padding: 0.75rem;
		border-bottom: 1px solid #334155;
	}

	th {
		color: #64748b;
		font-weight: 600;
	}

	.team-name {
		font-weight: 600;
	}

	.points {
		font-weight: bold;
		color: #38bdf8;
	}

	.news-list {
		list-style: none;
		padding: 0;
	}

	.news-list li {
		margin-bottom: 1.5rem;
		padding-bottom: 1rem;
		border-bottom: 1px solid #334155;
	}

	.news-list a {
		text-decoration: none;
		color: #f8fafc;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.news-list a:hover .news-title {
		color: #38bdf8;
	}

	.news-date {
		font-size: 0.75rem;
		color: #64748b;
	}

	.news-title {
		font-weight: bold;
		font-size: 1.1rem;
	}

	.news-list p {
		font-size: 0.9rem;
		color: #94a3b8;
		margin-top: 0.5rem;
	}

	.match-list {
		list-style: none;
		padding: 0;
	}

	.match-list li {
		padding: 0.75rem 0;
		border-bottom: 1px solid #334155;
	}

	.match-date {
		font-size: 0.75rem;
		color: #64748b;
		display: block;
		margin-bottom: 0.25rem;
	}

	.match-teams {
		display: flex;
		justify-content: space-between;
		font-weight: 500;
	}

	.match-result {
		background: #334155;
		padding: 0 0.5rem;
		border-radius: 0.25rem;
		color: #38bdf8;
	}

	.stat-list {
		list-style: none;
		padding: 0;
	}

	.stat-list li {
		display: flex;
		justify-content: space-between;
		padding: 0.5rem 0;
		border-bottom: 1px solid #334155;
	}

	.player {
		font-weight: 500;
	}

	.value {
		color: #38bdf8;
		font-weight: 600;
	}

	.mt-4 {
		margin-top: 2rem;
	}

	.error {
		color: #ef4444;
	}
</style>
