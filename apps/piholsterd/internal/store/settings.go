package store

import (
	"database/sql"
	"errors"
)

func (s *Store) Get(key string) (string, error) {
	var value string
	err := s.db.QueryRow(
		`SELECT value FROM settings WHERE key = ?`, key,
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", sql.ErrNoRows
	}
	return value, err
}

func (s *Store) Set(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			value      = excluded.value,
			updated_at = CURRENT_TIMESTAMP
	`, key, value)
	return err
}

// GetOrDefault returns the stored value for key, or defaultValue when the
// key does not exist or any error occurs.
func (s *Store) GetOrDefault(key, defaultValue string) string {
	v, err := s.Get(key)
	if err != nil {
		return defaultValue
	}
	return v
}
