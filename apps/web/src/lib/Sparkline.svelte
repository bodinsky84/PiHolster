<script>
	import { onMount, onDestroy } from 'svelte';
	import uPlot from 'uplot';
	import 'uplot/dist/uPlot.min.css';

	/** @type {{ts:string, total:number, blocked:number}[]} */
	export let buckets = [];
	export let height = 120;
	/** When true, render with axes and legend; otherwise a tight sparkline. */
	export let detailed = false;

	let el;
	let plot;

	function toData(bs) {
		const t = bs.map((b) => Math.floor(new Date(b.ts).getTime() / 1000));
		const total = bs.map((b) => b.total);
		const blocked = bs.map((b) => b.blocked);
		const allowed = bs.map((b) => Math.max(0, b.total - b.blocked));
		return [t, total, blocked, allowed];
	}

	function makeOpts(width) {
		const baseSeries = [
			{},
			{ label: 'Totalt', stroke: '#7eb8f7', width: 2, points: { show: false } },
			{ label: 'Blockerat', stroke: '#e74c3c', width: 2, points: { show: false } },
			{ label: 'Tillåtet', stroke: '#52b788', width: 1.25, points: { show: false } }
		];
		return {
			width,
			height,
			cursor: detailed ? { drag: { x: true, y: false } } : { show: false },
			legend: { show: detailed },
			scales: { x: { time: true } },
			axes: detailed
				? [
						{ stroke: '#888', grid: { stroke: '#2a2a3e' } },
						{ stroke: '#888', grid: { stroke: '#2a2a3e' } }
					]
				: [{ show: false }, { show: false }],
			series: baseSeries
		};
	}

	onMount(() => {
		const ro = new ResizeObserver(() => {
			if (plot) plot.setSize({ width: el.clientWidth, height });
		});
		ro.observe(el);

		plot = new uPlot(makeOpts(el.clientWidth), toData(buckets), el);
		return () => {
			ro.disconnect();
			plot?.destroy();
		};
	});

	$: if (plot && buckets) plot.setData(toData(buckets));

	onDestroy(() => plot?.destroy());
</script>

<div bind:this={el} class="spark"></div>

<style>
	.spark {
		width: 100%;
	}

	:global(.spark .u-legend) {
		color: #aaa;
		font-size: 0.78rem;
	}
</style>
