package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/piholster/piholster/apps/piholsterd/internal/store"
)

type DevicesHandler struct {
	store *store.Store
}

func NewDevicesHandler(st *store.Store) *DevicesHandler {
	return &DevicesHandler{store: st}
}

type deviceResponse struct {
	MAC         string    `json:"mac"`
	IP          string    `json:"ip"`
	Hostname    string    `json:"hostname"`
	DisplayName string    `json:"display_name"`
	Trusted     bool      `json:"trusted"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
}

func (h *DevicesHandler) List(w http.ResponseWriter, r *http.Request) {
	devices, err := h.store.ListDevices()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list devices failed"})
		return
	}

	resp := make([]deviceResponse, len(devices))
	for i, d := range devices {
		resp[i] = deviceResponse{
			MAC:         d.MAC,
			IP:          d.IP,
			Hostname:    d.Hostname,
			DisplayName: d.DisplayName,
			Trusted:     d.Trusted,
			FirstSeen:   d.FirstSeen,
			LastSeen:    d.LastSeen,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *DevicesHandler) SetTrust(w http.ResponseWriter, r *http.Request) {
	mac := r.PathValue("mac")
	if mac == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mac required"})
		return
	}

	var body struct {
		Trusted bool `json:"trusted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.store.SetDeviceTrusted(mac, body.Trusted); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DevicesHandler) Rename(w http.ResponseWriter, r *http.Request) {
	mac := r.PathValue("mac")
	if mac == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mac required"})
		return
	}

	var body struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if body.DisplayName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "display_name required"})
		return
	}

	if err := h.store.SetDeviceDisplayName(mac, body.DisplayName); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
