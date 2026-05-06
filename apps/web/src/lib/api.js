// Lightweight typed-via-JSDoc API client for the dashboard.
// Keeping this in plain JS (with JSDoc) avoids dragging in TypeScript tooling
// just for shared fetchers; SvelteKit handles JSDoc types natively.

/**
 * @typedef {Object} Bucket
 * @property {string} ts
 * @property {number} total
 * @property {number} blocked
 */

/**
 * @typedef {Object} TimeSeriesResponse
 * @property {number} window_seconds
 * @property {number} bucket_seconds
 * @property {Bucket[]} buckets
 */

/**
 * @typedef {Object} TopRow
 * @property {string} label
 * @property {number} count
 * @property {number} blocked
 */

/**
 * @typedef {Object} TopResponse
 * @property {string} kind
 * @property {number} window
 * @property {TopRow[]} rows
 */

/**
 * @typedef {Object} LatencyResponse
 * @property {number} window
 * @property {{p50:number, p95:number, p99:number, max:number, sample:number}} percentiles
 */

/**
 * @typedef {Object} SystemResponse
 * @property {number} uptime_seconds
 * @property {string} go_version
 * @property {number} goroutines
 * @property {number} heap_alloc_mb
 * @property {number} sys_mb
 * @property {number} num_gc
 */

/**
 * @typedef {Object} LiveEvent
 * @property {string} ts
 * @property {string} domain
 * @property {string} client_ip
 * @property {boolean} blocked
 * @property {string} upstream
 * @property {number} latency_ms
 */

async function getJSON(url) {
	const res = await fetch(url, { credentials: 'same-origin' });
	if (!res.ok) {
		throw new Error(`${url} → ${res.status}`);
	}
	return res.json();
}

/** @returns {Promise<TimeSeriesResponse>} */
export const fetchTimeseries = (window = '1h', bucket = '60s') =>
	getJSON(`/api/stats/timeseries?window=${window}&bucket=${bucket}`);

/** @returns {Promise<TopResponse>} */
export const fetchTop = (kind = 'blocked', limit = 10, window = '24h') =>
	getJSON(`/api/stats/top?kind=${kind}&limit=${limit}&window=${window}`);

/** @returns {Promise<TopResponse>} */
export const fetchClients = (limit = 10, window = '24h') =>
	getJSON(`/api/stats/clients?limit=${limit}&window=${window}`);

/** @returns {Promise<LatencyResponse>} */
export const fetchLatency = (window = '24h') =>
	getJSON(`/api/stats/latency?window=${window}`);

/** @returns {Promise<SystemResponse>} */
export const fetchSystem = () => getJSON('/api/stats/system');

/**
 * Open an SSE connection to the live query feed. Returns a teardown function.
 * @param {(e: LiveEvent) => void} onEvent
 * @param {{replay?: number}} [opts]
 * @returns {() => void}
 */
export function subscribeLive(onEvent, opts = {}) {
	const replay = opts.replay ?? 50;
	const es = new EventSource(`/api/stats/live?replay=${replay}`);
	es.onmessage = (m) => {
		try {
			onEvent(JSON.parse(m.data));
		} catch {
			// malformed line — ignore so a single bad event doesn't kill the stream
		}
	};
	return () => es.close();
}
