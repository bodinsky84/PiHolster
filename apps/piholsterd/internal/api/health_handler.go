package api

import (
	"net/http"
	"time"

	"github.com/piholster/piholster/apps/piholsterd/internal/store"
)

type HealthHandler struct {
	store *store.Store
}

func NewHealthHandler(st *store.Store) *HealthHandler {
	return &HealthHandler{store: st}
}

type healthResponse struct {
	Status      string `json:"status"`
	BlockedToday int64  `json:"blocked_today"`
	TotalToday   int64  `json:"total_today"`
	DNSRunning   bool   `json:"dns_running"`
}

// Health handles GET /api/health.
// Public endpoint — no auth required. Returns aggregate counters only;
// no device details, MAC addresses or IP addresses are exposed.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-24 * time.Hour)

	stats, err := h.store.QueryStats(since)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query stats failed"})
		return
	}

	writeJSON(w, http.StatusOK, healthResponse{
		Status:       "ok",
		BlockedToday: stats.Blocked,
		TotalToday:   stats.Total,
		DNSRunning:   true,
	})
}
