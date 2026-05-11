package income

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/piholster/piholster/apps/piholsterd/internal/store"
)

type IncomeService struct {
	store *store.Store
}

func NewIncomeService(st *store.Store) *IncomeService {
	return &IncomeService{store: st}
}

func (s *IncomeService) GetStats(w http.ResponseWriter, r *http.Request) {
	incomePerAdStr := s.store.GetOrDefault("income_per_ad", "0.10")
	incomePerAd, _ := strconv.ParseFloat(incomePerAdStr, 64)
	stats, _ := s.store.QueryStats(time.Time{})
	totalSaved := float64(stats.Blocked) * incomePerAd
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"total_saved": totalSaved,
		"blocked_total": stats.Blocked,
	})
}
