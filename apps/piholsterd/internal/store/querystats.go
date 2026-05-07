package store

import (
	"fmt"
	"time"
)

// Bucket aggregates query_log within a time window for sparkline rendering.
type Bucket struct {
	TS      time.Time `json:"ts"`
	Total   int64     `json:"total"`
	Blocked int64     `json:"blocked"`
}

// TopRow is a (label, count) pair for top-N domain/client lists.
type TopRow struct {
	Label   string `json:"label"`
	Count   int64  `json:"count"`
	Blocked int64  `json:"blocked"`
}

// LatencyPercentiles summarises upstream resolution latency in milliseconds.
// Computed only over non-blocked queries (blocked=NXDOMAIN, no upstream call).
type LatencyPercentiles struct {
	P50    int   `json:"p50"`
	P95    int   `json:"p95"`
	P99    int   `json:"p99"`
	Max    int   `json:"max"`
	Sample int64 `json:"sample"`
}

// TimeSeries returns one Bucket per bucketSeconds-wide window between since
// and now. Empty buckets get returned as zero counts so the frontend chart
// has a stable x-axis without holes.
func (s *Store) TimeSeries(since time.Time, bucketSeconds int) ([]Bucket, error) {
	if bucketSeconds < 1 {
		return nil, fmt.Errorf("bucketSeconds must be >= 1")
	}

	rows, err := s.db.Query(`
		SELECT
		    CAST(strftime('%s', queried_at) AS INTEGER) / ? * ? AS bucket_ts,
		    COUNT(*),
		    COALESCE(SUM(blocked), 0)
		FROM query_log
		WHERE queried_at >= ?
		GROUP BY bucket_ts
		ORDER BY bucket_ts ASC
	`, bucketSeconds, bucketSeconds, since.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byBucket := make(map[int64]Bucket)
	for rows.Next() {
		var bucketTS int64
		var b Bucket
		if err := rows.Scan(&bucketTS, &b.Total, &b.Blocked); err != nil {
			return nil, err
		}
		b.TS = time.Unix(bucketTS, 0).UTC()
		byBucket[bucketTS] = b
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	startBucket := since.UTC().Unix() / int64(bucketSeconds) * int64(bucketSeconds)
	endBucket := time.Now().UTC().Unix() / int64(bucketSeconds) * int64(bucketSeconds)
	out := make([]Bucket, 0, (endBucket-startBucket)/int64(bucketSeconds)+1)
	for ts := startBucket; ts <= endBucket; ts += int64(bucketSeconds) {
		if b, ok := byBucket[ts]; ok {
			out = append(out, b)
		} else {
			out = append(out, Bucket{TS: time.Unix(ts, 0).UTC()})
		}
	}
	return out, nil
}

// TopDomains returns the most frequently queried domains since the given time.
// kind: "blocked" → only blocked queries, "allowed" → only allowed, "" → all.
func (s *Store) TopDomains(since time.Time, kind string, limit int) ([]TopRow, error) {
	if limit < 1 {
		limit = 10
	}

	where := "queried_at >= ?"
	args := []any{since.UTC().Format("2006-01-02 15:04:05")}
	switch kind {
	case "blocked":
		where += " AND blocked = 1"
	case "allowed":
		where += " AND blocked = 0"
	}

	q := `
		SELECT domain, COUNT(*) AS n, COALESCE(SUM(blocked), 0)
		FROM query_log
		WHERE ` + where + `
		GROUP BY domain
		ORDER BY n DESC
		LIMIT ?
	`
	args = append(args, limit)

	return s.scanTopRows(q, args...)
}

// TopClients returns the most active clients (by IP) since the given time.
func (s *Store) TopClients(since time.Time, limit int) ([]TopRow, error) {
	if limit < 1 {
		limit = 10
	}
	return s.scanTopRows(`
		SELECT client_ip, COUNT(*) AS n, COALESCE(SUM(blocked), 0)
		FROM query_log
		WHERE queried_at >= ?
		GROUP BY client_ip
		ORDER BY n DESC
		LIMIT ?
	`, since.UTC().Format("2006-01-02 15:04:05"), limit)
}

func (s *Store) scanTopRows(query string, args ...any) ([]TopRow, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TopRow
	for rows.Next() {
		var r TopRow
		if err := rows.Scan(&r.Label, &r.Count, &r.Blocked); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Latency returns p50/p95/p99/max for non-blocked queries since the given time.
// Implemented in Go after fetching all latencies because SQLite lacks
// percentile_cont. Sample size is capped at 50000 to bound memory on a Pi.
func (s *Store) Latency(since time.Time) (LatencyPercentiles, error) {
	rows, err := s.db.Query(`
		SELECT latency_ms FROM query_log
		WHERE queried_at >= ? AND blocked = 0 AND latency_ms > 0
		ORDER BY latency_ms ASC
		LIMIT 50000
	`, since.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		return LatencyPercentiles{}, err
	}
	defer rows.Close()

	var lat []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return LatencyPercentiles{}, err
		}
		lat = append(lat, v)
	}
	if err := rows.Err(); err != nil {
		return LatencyPercentiles{}, err
	}

	n := len(lat)
	if n == 0 {
		return LatencyPercentiles{}, nil
	}
	pick := func(p float64) int {
		idx := int(float64(n-1) * p)
		return lat[idx]
	}
	return LatencyPercentiles{
		P50:    pick(0.50),
		P95:    pick(0.95),
		P99:    pick(0.99),
		Max:    lat[n-1],
		Sample: int64(n),
	}, nil
}
