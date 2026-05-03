package api

import (
	"net/http"
	"time"

	"github.com/piholster/piholster/apps/piholsterd/internal/store"
)

type StatusHandler struct {
	store *store.Store
}

func NewStatusHandler(st *store.Store) *StatusHandler {
	return &StatusHandler{store: st}
}

type statusResponse struct {
	Status        string `json:"status"`
	BlockedToday  int64  `json:"blocked_today"`
	TotalToday    int64  `json:"total_today"`
	DevicesOnline int    `json:"devices_online"`
	DNSRunning    bool   `json:"dns_running"`
}

func (h *StatusHandler) Status(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-24 * time.Hour)

	stats, err := h.store.QueryStats(since)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query stats failed"})
		return
	}

	devices, err := h.store.ListDevices()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list devices failed"})
		return
	}

	onlineThreshold := time.Now().Add(-5 * time.Minute)
	online := 0
	for _, d := range devices {
		if d.LastSeen.After(onlineThreshold) {
			online++
		}
	}

	status := "ok"
	for _, d := range devices {
		if !d.Trusted && d.LastSeen.After(onlineThreshold) {
			status = "warning"
			break
		}
	}

	writeJSON(w, http.StatusOK, statusResponse{
		Status:        status,
		BlockedToday:  stats.Blocked,
		TotalToday:    stats.Total,
		DevicesOnline: online,
		DNSRunning:    true,
	})
}
