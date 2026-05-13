package allsvenskan

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type StandingsEntry struct {
	Position int    `json:"position"`
	Team     string `json:"team"`
	Games    int    `json:"games"`
	Wins     int    `json:"wins"`
	Draws    int    `json:"draws"`
	Losses   int    `json:"losses"`
	Goals    string `json:"goals"`
	Points   int    `json:"points"`
}

type Match struct {
	Date     string `json:"date"`
	HomeTeam string `json:"home_team"`
	AwayTeam string `json:"away_team"`
	Result   string `json:"result"`
}

type NewsItem struct {
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Description string    `json:"description"`
	PubDate     time.Time `json:"pub_date"`
}

type Stats struct {
	TopScorers []PlayerStat `json:"top_scorers"`
	TopCards   []PlayerStat `json:"top_cards"`
}

type PlayerStat struct {
	Name  string `json:"name"`
	Team  string `json:"team"`
	Value int    `json:"value"`
}

type Engine struct {
	standings []StandingsEntry
	matches   []Match
	news      []NewsItem
	stats     Stats
	mu        sync.RWMutex
}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Run(ctx context.Context) {
	slog.Info("allsvenskan: Engine started")
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	// Initial fetch
	e.update(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.update(ctx)
		}
	}
}

func (e *Engine) update(ctx context.Context) {
	news := ScrapeNews(ctx)
	standings := ScrapeStandings(ctx)
	matches := ScrapeMatches(ctx)
	stats := ScrapeStats(ctx)

	e.mu.Lock()
	defer e.mu.Unlock()

	if len(news) > 0 {
		e.news = news
	}
	if len(standings) > 0 {
		e.standings = standings
	}
	if len(matches) > 0 {
		e.matches = matches
	}
	if len(stats.TopScorers) > 0 {
		e.stats = stats
	}
}

func (e *Engine) GetStandings() []StandingsEntry {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.standings
}

func (e *Engine) GetNews() []NewsItem {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.news
}

func (e *Engine) GetMatches() []Match {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.matches
}

func (e *Engine) GetStats() Stats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stats
}
