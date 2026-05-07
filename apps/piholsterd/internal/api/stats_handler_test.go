package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/piholster/piholster/apps/piholsterd/internal/queryevents"
	"github.com/piholster/piholster/apps/piholsterd/internal/store"
)

// flushRecorder wraps httptest.ResponseRecorder with a no-op Flush so handlers
// that require http.Flusher (the SSE handler) can run inside tests.
type flushRecorder struct {
	*httptest.ResponseRecorder
}

func (f *flushRecorder) Flush() {}

func newTestStatsHandler(t *testing.T) (*StatsHandler, *store.Store, *queryevents.Bus) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	bus := queryevents.NewBus(16)
	return NewStatsHandler(st, bus), st, bus
}

func seedQueries(t *testing.T, st *store.Store) {
	t.Helper()
	rows := []struct {
		domain   string
		clientIP string
		blocked  bool
		latency  int
	}{
		{"ads.example.com", "10.0.0.1", true, 0},
		{"ads.example.com", "10.0.0.1", true, 0},
		{"ads.example.com", "10.0.0.2", true, 0},
		{"news.example.com", "10.0.0.1", false, 12},
		{"news.example.com", "10.0.0.2", false, 25},
		{"api.example.com", "10.0.0.3", false, 50},
	}
	for _, r := range rows {
		if err := st.LogQuery(r.domain, r.clientIP, r.blocked, "doh", r.latency); err != nil {
			t.Fatalf("LogQuery: %v", err)
		}
	}
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, rec.Body.String())
	}
	return out
}

func TestTimeSeriesReturnsBuckets(t *testing.T) {
	h, st, _ := newTestStatsHandler(t)
	seedQueries(t, st)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/timeseries?window=1h&bucket=60s", nil)
	rec := httptest.NewRecorder()
	h.TimeSeries(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSON(t, rec)
	if body["bucket_seconds"].(float64) != 60 {
		t.Errorf("bucket_seconds = %v, want 60", body["bucket_seconds"])
	}
	if _, ok := body["buckets"].([]any); !ok {
		t.Errorf("buckets missing or not an array: %T", body["buckets"])
	}
}

func TestTopBlockedDomains(t *testing.T) {
	h, st, _ := newTestStatsHandler(t)
	seedQueries(t, st)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/top?kind=blocked&limit=5&window=24h", nil)
	rec := httptest.NewRecorder()
	h.Top(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := decodeJSON(t, rec)
	rows, ok := body["rows"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatalf("rows missing or empty: %v", body["rows"])
	}
	first := rows[0].(map[string]any)
	if first["label"].(string) != "ads.example.com" {
		t.Errorf("top blocked label = %v, want ads.example.com", first["label"])
	}
	if first["count"].(float64) != 3 {
		t.Errorf("top blocked count = %v, want 3", first["count"])
	}
}

func TestClientsRanksByQueryCount(t *testing.T) {
	h, st, _ := newTestStatsHandler(t)
	seedQueries(t, st)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/clients?limit=5&window=24h", nil)
	rec := httptest.NewRecorder()
	h.Clients(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := decodeJSON(t, rec)
	rows := body["rows"].([]any)
	if len(rows) == 0 {
		t.Fatal("rows empty")
	}
	first := rows[0].(map[string]any)
	if first["label"].(string) != "10.0.0.1" {
		t.Errorf("top client = %v, want 10.0.0.1", first["label"])
	}
}

func TestLatencyComputesPercentiles(t *testing.T) {
	h, st, _ := newTestStatsHandler(t)
	seedQueries(t, st)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/latency?window=24h", nil)
	rec := httptest.NewRecorder()
	h.Latency(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := decodeJSON(t, rec)
	p := body["percentiles"].(map[string]any)
	if p["sample"].(float64) != 3 {
		t.Errorf("sample = %v, want 3 (non-blocked, latency>0)", p["sample"])
	}
	if p["max"].(float64) != 50 {
		t.Errorf("max = %v, want 50", p["max"])
	}
}

func TestSystemReturnsRuntimeInfo(t *testing.T) {
	h, _, _ := newTestStatsHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/system", nil)
	rec := httptest.NewRecorder()
	h.System(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := decodeJSON(t, rec)
	for _, k := range []string{"uptime_seconds", "go_version", "goroutines", "heap_alloc_mb", "sys_mb", "num_gc"} {
		if _, ok := body[k]; !ok {
			t.Errorf("system response missing key %q", k)
		}
	}
}

// TestLiveReplaysRecentEvents verifies that connecting to /api/stats/live with
// ?replay=N immediately gets the most recent N events from the bus, then the
// handler returns when the request context is cancelled.
func TestLiveReplaysRecentEvents(t *testing.T) {
	h, _, bus := newTestStatsHandler(t)

	bus.Publish(queryevents.Event{Timestamp: time.Now(), Domain: "first.example.com"})
	bus.Publish(queryevents.Event{Timestamp: time.Now(), Domain: "second.example.com"})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/stats/live?replay=10", nil).WithContext(ctx)
	rec := &flushRecorder{httptest.NewRecorder()}
	h.Live(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "first.example.com") {
		t.Errorf("body missing first event; got: %s", body)
	}
	if !strings.Contains(body, "second.example.com") {
		t.Errorf("body missing second event; got: %s", body)
	}
	if rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", rec.Header().Get("Content-Type"))
	}
}

