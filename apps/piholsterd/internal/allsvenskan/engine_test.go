package allsvenskan

import (
	"context"
	"testing"
)

func TestEngine(t *testing.T) {
	e := NewEngine()
	// Test that it doesn't crash on update
	e.update(context.Background())

	e.GetStandings()
	e.GetNews()
	e.GetMatches()
	e.GetPlayers()
}

func TestScrapers(t *testing.T) {
	ctx := context.Background()

	ScrapeNews(ctx)
	ScrapeStandings(ctx)
	ScrapeMatches(ctx)
	ScrapePlayerMarketData(ctx)
}
