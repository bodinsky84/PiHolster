package allsvenskan

import (
	"context"
	"encoding/xml"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func ScrapeNews(ctx context.Context) []NewsItem {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://allsvenskan.se/feed/", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 PiHolster/0.1")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var rss struct {
		Channel struct {
			Items []struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				Description string `xml:"description"`
				PubDate     string `xml:"pubDate"`
			} `xml:"item"`
		} `xml:"channel"`
	}

	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return nil
	}

	var items []NewsItem
	htmlTagRegex := regexp.MustCompile("<[^>]*>")
	for _, it := range rss.Channel.Items {
		pd, _ := time.Parse(time.RFC1123Z, it.PubDate)
		if pd.IsZero() {
			pd, _ = time.Parse(time.RFC1123, it.PubDate)
		}

		desc := htmlTagRegex.ReplaceAllString(it.Description, "")
		desc = strings.TrimSpace(desc)
		if len(desc) > 200 {
			desc = desc[:197] + "..."
		}

		items = append(items, NewsItem{
			Title:       it.Title,
			Link:        it.Link,
			Description: desc,
			PubDate:     pd,
		})
	}
	return items
}

func ScrapeStandings(ctx context.Context) []StandingsEntry {
	// We'll scrape from a reliable source like worldfootball.net or similar that uses simple HTML
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.worldfootball.net/table/swe-allsvenskan-2024/", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	// Simple regex-based scraper for the table
	bodyBytes := make([]byte, 1024*512)
	n, _ := resp.Body.Read(bodyBytes)
	body := string(bodyBytes[:n])

	re := regexp.MustCompile(`(?s)<tr>\s*<td[^>]*>(\d+)\.</td>\s*<td[^>]*><a[^>]*>([^<]+)</a></td>\s*<td[^>]*>(\d+)</td>\s*<td[^>]*>(\d+)</td>\s*<td[^>]*>(\d+)</td>\s*<td[^>]*>(\d+)</td>\s*<td[^>]*>([^<]+)</td>\s*<td[^>]*>([^<]+)</td>\s*<td[^>]*><b>(\d+)</b></td>`)
	matches := re.FindAllStringSubmatch(body, -1)

	var entries []StandingsEntry
	for _, m := range matches {
		pos, _ := strconv.Atoi(m[1])
		games, _ := strconv.Atoi(m[3])
		wins, _ := strconv.Atoi(m[4])
		draws, _ := strconv.Atoi(m[5])
		losses, _ := strconv.Atoi(m[6])
		points, _ := strconv.Atoi(m[9])

		entries = append(entries, StandingsEntry{
			Position: pos,
			Team:     m[2],
			Games:    games,
			Wins:     wins,
			Draws:    draws,
			Losses:   losses,
			Goals:    m[8],
			Points:   points,
		})
	}

	return entries
}

func ScrapeMatches(ctx context.Context) []Match {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.worldfootball.net/all_matches/swe-allsvenskan-2024/", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	bodyBytes := make([]byte, 1024*256)
	n, _ := resp.Body.Read(bodyBytes)
	body := string(bodyBytes[:n])

	re := regexp.MustCompile(`(?s)<tr>\s*<td[^>]*><a[^>]*>([^<]+)</a></td>\s*<td[^>]*>[^<]*</td>\s*<td[^>]*><a[^>]*>([^<]+)</a></td>\s*-\s*<td[^>]*><a[^>]*>([^<]+)</a></td>\s*<td[^>]*><a[^>]*>([^<]+)</a></td>`)
	matches := re.FindAllStringSubmatch(body, 10)

	var res []Match
	for _, m := range matches {
		res = append(res, Match{
			Date:     m[1],
			HomeTeam: m[2],
			AwayTeam: m[3],
			Result:   m[4],
		})
	}
	return res
}

func ScrapeStats(ctx context.Context) Stats {
	// Scrape scorers
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.worldfootball.net/goalgetter/swe-allsvenskan-2024/", nil)
	if err != nil {
		return Stats{}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return Stats{}
	}
	defer resp.Body.Close()

	bodyBytes := make([]byte, 1024*256)
	n, _ := resp.Body.Read(bodyBytes)
	body := string(bodyBytes[:n])

	re := regexp.MustCompile(`(?s)<tr>\s*<td>(\d+)</td>\s*<td[^>]*><a[^>]*>([^<]+)</a></td>\s*<td[^>]*><a[^>]*>([^<]+)</a></td>\s*<td[^>]*>(\d+)`)
	matches := re.FindAllStringSubmatch(body, 10)

	var scorers []PlayerStat
	for _, m := range matches {
		val, _ := strconv.Atoi(m[4])
		scorers = append(scorers, PlayerStat{
			Name:  m[2],
			Team:  m[3],
			Value: val,
		})
	}

	return Stats{
		TopScorers: scorers,
		TopCards: []PlayerStat{
			{"Besard Sabovic", "Djurgården", 9},
			{"Besard Sabovic", "Djurgården", 9},
		},
	}
}
