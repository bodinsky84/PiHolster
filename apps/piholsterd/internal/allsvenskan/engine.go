package allsvenskan

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Source represents a data origin with its metadata
type Source struct {
	Name             string  `json:"name"`
	Domain           string  `json:"domain"`
	SourceType       string  `json:"source_type"` // "OFFICIAL", "API", "MEDIA", "SOCIAL"
	ReliabilityScore float64 `json:"reliability_score"`
	UpdateFrequency  string  `json:"update_frequency"`
}

// Player represents a professional football player
type Player struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Team           string            `json:"team"`
	Position       string            `json:"position"`
	Age            int               `json:"age"`
	Foot           string            `json:"foot"`
	MarketValue    string            `json:"market_value"`
	AcquisitionFee string            `json:"acquisition_fee"`
	Stats          PlayerStats       `json:"stats"`
	RadarValues    map[string]float64 `json:"radar_values"` // Normalized 0-100
}

// PlayerStats contains performance metrics
type PlayerStats struct {
	MatchesPlayed int     `json:"matches_played"`
	Minutes       int     `json:"minutes"`
	Goals         int     `json:"goals"`
	Assists       int     `json:"assists"`
	XG            float64 `json:"xg"`
	XA            float64 `json:"xa"`
	Shots         int     `json:"shots"`
	Passes        int     `json:"passes"`
	Tackles       int     `json:"tackles"`
	Interceptions int     `json:"interceptions"`
	DuelsWon      int     `json:"duels_won"`
	YellowCards   int     `json:"yellow_cards"`
	RedCards      int     `json:"red_cards"`
}

type Match struct {
	ID       string    `json:"id"`
	Date     time.Time `json:"date"`
	HomeTeam string    `json:"home_team"`
	AwayTeam string    `json:"away_team"`
	Result   string    `json:"result"`
	Status   string    `json:"status"` // "SCHEDULED", "LIVE", "FINISHED"
}

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

type Transfer struct {
	PlayerName string    `json:"player_name"`
	FromTeam   string    `json:"from_team"`
	ToTeam     string    `json:"to_team"`
	Fee        string    `json:"fee"`
	Date       time.Time `json:"date"`
	IsRumour   bool      `json:"is_rumour"`
}

type NewsItem struct {
	Source      string    `json:"source"`
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Description string    `json:"description"`
	PubDate     time.Time `json:"pub_date"`
}

type Engine struct {
	registry   []Source
	standings  []StandingsEntry
	matches    []Match
	players    []Player
	news       []NewsItem
	transfers  []Transfer
	mu         sync.RWMutex
}

func NewEngine() *Engine {
	return &Engine{
		registry: []Source{
			{"SvFF", "svenskfotboll.se", "OFFICIAL", 1.0, "Daily"},
			{"Allsvenskan", "allsvenskan.se", "OFFICIAL", 1.0, "Hourly"},
			{"Transfermarkt", "transfermarkt.com", "MEDIA", 0.85, "Daily"},
			{"Expressen", "expressen.se", "MEDIA", 0.8, "Hourly"},
			{"Fotbollskanalen", "fotbollskanalen.se", "MEDIA", 0.8, "Hourly"},
		},
	}
}

func (e *Engine) Run(ctx context.Context) {
	slog.Info("allsvenskan: engine started")
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

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
	e.mu.Lock()
	defer e.mu.Unlock()

	e.news = ScrapeNews(ctx)
	e.standings = ScrapeStandings(ctx)
	e.matches = ScrapeMatches(ctx)
	e.players = ScrapePlayerMarketData(ctx)

	// Post-processing: Normalize radar values
	for i := range e.players {
		e.players[i].RadarValues = calculateRadarValues(e.players[i])
	}
}

func calculateRadarValues(p Player) map[string]float64 {
	// Normalization logic for spider charts
	// Using placeholder math based on available stats
	return map[string]float64{
		"Attack":     normalize(float64(p.Stats.Goals), 0, 15),
		"Creativity": normalize(p.Stats.XA, 0, 10),
		"Possession": normalize(float64(p.Stats.Passes), 0, 1000),
		"Defense":    normalize(float64(p.Stats.Tackles), 0, 50),
		"Physical":   80.0, // Placeholder
	}
}

func normalize(val, min, max float64) float64 {
	if max == min {
		return 0
	}
	res := ((val - min) / (max - min)) * 100
	if res > 100 {
		return 100
	}
	if res < 0 {
		return 0
	}
	return res
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

func (e *Engine) GetPlayers() []Player {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.players
}

type AggregatedStats struct {
	TopScorers []StatEntry `json:"top_scorers"`
	TopCards   []StatEntry `json:"top_cards"`
}

type StatEntry struct {
	Name  string `json:"name"`
	Team  string `json:"team"`
	Value int    `json:"value"`
}

func (e *Engine) GetStats() AggregatedStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// In a real scenario, we'd sort e.players by goals/cards
	// Using mock logic for now since scraper only gets market values
	return AggregatedStats{
		TopScorers: []StatEntry{
			{"Ioannis Pittas", "AIK", 7},
			{"Erik Botheim", "Malmö FF", 6},
		},
		TopCards: []StatEntry{
			{"Besard Sabovic", "Djurgården", 4},
			{"Anton Tinnerholm", "Malmö FF", 4},
		},
	}
}
