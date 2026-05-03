package store

import "time"

type Stats struct {
	Total   int64
	Blocked int64
}

func (s *Store) LogQuery(domain, clientIP string, blocked bool, upstream string, latencyMs int) error {
	b := 0
	if blocked {
		b = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO query_log (domain, client_ip, blocked, upstream, latency_ms)
		VALUES (?, ?, ?, ?, ?)
	`, domain, clientIP, b, upstream, latencyMs)
	return err
}

// QueryStats returns totals for all queries since the given time.
func (s *Store) QueryStats(since time.Time) (Stats, error) {
	row := s.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(blocked), 0)
		FROM query_log
		WHERE queried_at >= ?
	`, since.UTC().Format("2006-01-02 15:04:05"))

	var st Stats
	err := row.Scan(&st.Total, &st.Blocked)
	return st, err
}

// PruneOldLogs deletes query_log rows whose queried_at is older than the
// given duration relative to now.
func (s *Store) PruneOldLogs(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan).UTC().Format("2006-01-02 15:04:05")
	_, err := s.db.Exec(
		`DELETE FROM query_log WHERE queried_at < ?`,
		cutoff,
	)
	return err
}
