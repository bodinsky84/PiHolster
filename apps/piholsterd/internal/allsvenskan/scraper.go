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
	sources := map[string]string{
		"Allsvenskan":     "https://allsvenskan.se/feed/",
		"Expressen":      "https://www.expressen.se/rss/sport/fotboll/allsvenskan/",
		"Fotbollskanalen": "https://www.fotbollskanalen.se/allsvenskan/rss",
	}

	var allNews []NewsItem
	htmlTagRegex := regexp.MustCompile("<[^>]*>")
	client := &http.Client{Timeout: 10 * time.Second}

	for name, url := range sources {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

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
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

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

			allNews = append(allNews, NewsItem{
				Source:      name,
				Title:       it.Title,
				Link:        it.Link,
				Description: desc,
				PubDate:     pd,
			})
		}
	}
	return allNews
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
		d, _ := time.Parse("02/01/2006", m[1])
		res = append(res, Match{
			Date:     d,
			HomeTeam: m[2],
			AwayTeam: m[3],
			Result:   m[4],
		})
	}
	return res
}

func ScrapePlayerMarketData(ctx context.Context) []Player {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.transfermarkt.com/allsvenskan/marktwerte/wettbewerb/SE1", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	bodyBytes := make([]byte, 1024*512)
	n, _ := resp.Body.Read(bodyBytes)
	body := string(bodyBytes[:n])

	re := regexp.MustCompile(`(?s)<td class="hauptlink">.*?<a title="(.*?)" href="(.*?)">.*?<\/a>.*?<\/td>.*?<a title="(.*?)" href=".*?">.*?<\/a>.*?<td class="rechts hauptlink"><a.*?>(.*?)<\/a>`)
	matches := re.FindAllStringSubmatch(body, 25)

	var players []Player
	for _, m := range matches {
		players = append(players, Player{
			Name:        m[1],
			Team:        m[3],
			MarketValue: m[4],
			ID:          m[2],
		})
	}
	return players
}
