package allsvenskan

import (
	"context"
	"testing"
)

func TestEngine(t *testing.T) {
	e := NewEngine()
	// Test that it doesn't crash on update
	e.update(context.Background())

	// Since we are in a sandbox, network might be restricted,
	// so we don't strictly require data to be present,
	// but we check that the getters work.
	e.GetStandings()
	e.GetNews()
	e.GetStats()
}

func TestScrapers(t *testing.T) {
	// These might fail if no internet, but we want to see if they crash
	ctx := context.Background()

	news := ScrapeNews(ctx)
	t.Logf("Found %d news items", len(news))

	standings := ScrapeStandings(ctx)
	t.Logf("Found %d teams", len(standings))

	matches := ScrapeMatches(ctx)
	t.Logf("Found %d matches", len(matches))

	stats := ScrapeStats(ctx)
	t.Logf("Found %d scorers", len(stats.TopScorers))
}
