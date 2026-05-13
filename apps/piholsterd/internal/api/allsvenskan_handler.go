package api

import (
	"net/http"

	"github.com/piholster/piholster/apps/piholsterd/internal/allsvenskan"
)

type AllsvenskanHandler struct {
	engine *allsvenskan.Engine
}

func NewAllsvenskanHandler(engine *allsvenskan.Engine) *AllsvenskanHandler {
	return &AllsvenskanHandler{engine: engine}
}

func (h *AllsvenskanHandler) Table(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.engine.GetStandings())
}

func (h *AllsvenskanHandler) News(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.engine.GetNews())
}

func (h *AllsvenskanHandler) Matches(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.engine.GetMatches())
}

func (h *AllsvenskanHandler) Stats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.engine.GetStats())
}
