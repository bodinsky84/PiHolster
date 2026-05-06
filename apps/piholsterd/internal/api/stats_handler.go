package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/piholster/piholster/apps/piholsterd/internal/queryevents"
	"github.com/piholster/piholster/apps/piholsterd/internal/store"
)

type StatsHandler struct {
	store     *store.Store
	bus       *queryevents.Bus
	startedAt time.Time
}

func NewStatsHandler(st *store.Store, bus *queryevents.Bus) *StatsHandler {
	return &StatsHandler{
		store:     st,
		bus:       bus,
		startedAt: time.Now(),
	}
}

func parseDurationParam(r *http.Request, name string, def time.Duration) time.Duration {
	v := r.URL.Query().Get(name)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func parseIntParam(r *http.Request, name string, def, min, max int) int {
	v := r.URL.Query().Get(name)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

// GET /api/stats/timeseries?window=1h&bucket=60s
func (h *StatsHandler) TimeSeries(w http.ResponseWriter, r *http.Request) {
	window := parseDurationParam(r, "window", time.Hour)
	bucket := parseDurationParam(r, "bucket", time.Minute)
	if bucket < time.Second {
		bucket = time.Minute
	}

	buckets, err := h.store.TimeSeries(time.Now().Add(-window), int(bucket.Seconds()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "timeseries failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"window_seconds": int(window.Seconds()),
		"bucket_seconds": int(bucket.Seconds()),
		"buckets":        buckets,
	})
}

// GET /api/stats/top?kind=blocked&limit=10&window=24h
func (h *StatsHandler) Top(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	limit := parseIntParam(r, "limit", 10, 1, 100)
	window := parseDurationParam(r, "window", 24*time.Hour)

	rows, err := h.store.TopDomains(time.Now().Add(-window), kind, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "top failed"})
		return
	}
	if rows == nil {
		rows = []store.TopRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"kind":   kind,
		"window": int(window.Seconds()),
		"rows":   rows,
	})
}

// GET /api/stats/clients?limit=10&window=24h
func (h *StatsHandler) Clients(w http.ResponseWriter, r *http.Request) {
	limit := parseIntParam(r, "limit", 10, 1, 100)
	window := parseDurationParam(r, "window", 24*time.Hour)

	rows, err := h.store.TopClients(time.Now().Add(-window), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "clients failed"})
		return
	}
	if rows == nil {
		rows = []store.TopRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"window": int(window.Seconds()),
		"rows":   rows,
	})
}

// GET /api/stats/latency?window=24h
func (h *StatsHandler) Latency(w http.ResponseWriter, r *http.Request) {
	window := parseDurationParam(r, "window", 24*time.Hour)
	p, err := h.store.Latency(time.Now().Add(-window))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "latency failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"window":      int(window.Seconds()),
		"percentiles": p,
	})
}

// GET /api/stats/system — runtime info: uptime, RSS estimate via runtime.MemStats,
// goroutines, Go version. Cheap and intentionally lightweight.
func (h *StatsHandler) System(w http.ResponseWriter, r *http.Request) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	writeJSON(w, http.StatusOK, map[string]any{
		"uptime_seconds": int64(time.Since(h.startedAt).Seconds()),
		"go_version":     runtime.Version(),
		"goroutines":     runtime.NumGoroutine(),
		"heap_alloc_mb":  ms.HeapAlloc / (1024 * 1024),
		"sys_mb":         ms.Sys / (1024 * 1024),
		"num_gc":         ms.NumGC,
	})
}

// GET /api/stats/live — Server-Sent Events stream of query events.
// On connect, replays up to ?replay=N (default 50, max 500) recent events.
func (h *StatsHandler) Live(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	replay := parseIntParam(r, "replay", 50, 0, 500)
	if replay > 0 {
		for _, e := range h.bus.Recent(replay) {
			writeSSEEvent(w, flusher, e)
		}
	}

	ch, cancel := h.bus.Subscribe(64)
	defer cancel()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case e, ok := <-ch:
			if !ok {
				return
			}
			writeSSEEvent(w, flusher, e)
		case <-heartbeat.C:
			fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func writeSSEEvent(w http.ResponseWriter, f http.Flusher, e queryevents.Event) {
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	f.Flush()
}
