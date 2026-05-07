<script>
	/** @type {{label:string, count:number, blocked:number}[]} */
	export let rows = [];
	export let title = '';
	export let valueLabel = '';
	export let monoLabel = false;

	$: max = rows.reduce((m, r) => Math.max(m, r.count), 0) || 1;
	$: total = rows.reduce((s, r) => s + r.count, 0);
</script>

<section class="card">
	<header class="card-head">
		<h2>{title}</h2>
		{#if total > 0}
			<span class="muted">{total.toLocaleString('sv-SE')}</span>
		{/if}
	</header>

	{#if rows.length === 0}
		<p class="muted">Inga data ännu.</p>
	{:else}
		<ul class="rows">
			{#each rows as r (r.label)}
				<li class="row">
					<span class="lbl" class:mono={monoLabel} title={r.label}>{r.label}</span>
					<span class="bar-wrap" aria-hidden="true">
						<span class="bar" style="width:{(r.count / max) * 100}%"></span>
					</span>
					<span class="num">{r.count.toLocaleString('sv-SE')}</span>
				</li>
			{/each}
		</ul>
	{/if}
	{#if valueLabel}
		<p class="hint muted">{valueLabel}</p>
	{/if}
</section>

<style>
	.card {
		background: #1a1a2e;
		border-radius: 0.75rem;
		padding: 1.25rem 1.5rem;
	}

	.card-head {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		margin-bottom: 0.75rem;
	}

	h2 {
		font-size: 0.85rem;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: #aaa;
		margin: 0;
	}

	.rows {
		list-style: none;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		padding: 0;
		margin: 0;
	}

	.row {
		display: grid;
		grid-template-columns: 1fr 80px 60px;
		align-items: center;
		gap: 0.6rem;
	}

	.lbl {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		font-size: 0.92rem;
	}

	.lbl.mono {
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		font-size: 0.85rem;
	}

	.bar-wrap {
		background: #0f0f1a;
		height: 6px;
		border-radius: 3px;
		overflow: hidden;
	}

	.bar {
		display: block;
		height: 100%;
		background: linear-gradient(90deg, #7eb8f7, #b08af7);
		border-radius: 3px;
	}

	.num {
		text-align: right;
		font-variant-numeric: tabular-nums;
		color: #d0d0d0;
		font-size: 0.9rem;
	}

	.muted {
		color: #888;
		font-size: 0.85rem;
	}

	.hint {
		margin: 0.75rem 0 0;
		font-size: 0.78rem;
	}

	@media (max-width: 480px) {
		.row {
			grid-template-columns: 1fr 50px 50px;
			gap: 0.4rem;
		}
	}
</style>
