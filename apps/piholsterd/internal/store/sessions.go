package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Session mirrors the sessions row returned from the database.
type Session struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
	LastSeen  time.Time
}

// CreateSession inserts a new session row.
func (s *Store) CreateSession(token string, userID int64, expiresAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("store: create session: %w", err)
	}
	return nil
}

// GetSession returns the session, or nil when the token does not exist or has
// expired. Expired rows are not deleted here; use PruneSessions for that.
func (s *Store) GetSession(token string) (*Session, error) {
	sess := &Session{}
	err := s.db.QueryRow(
		`SELECT token, user_id, expires_at, last_seen
		 FROM sessions
		 WHERE token = ? AND expires_at > CURRENT_TIMESTAMP`,
		token,
	).Scan(&sess.Token, &sess.UserID, &sess.ExpiresAt, &sess.LastSeen)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get session: %w", err)
	}
	return sess, nil
}

// TouchSession updates the last_seen timestamp, extending idle visibility.
func (s *Store) TouchSession(token string) error {
	_, err := s.db.Exec(
		`UPDATE sessions SET last_seen = CURRENT_TIMESTAMP WHERE token = ?`,
		token,
	)
	if err != nil {
		return fmt.Errorf("store: touch session: %w", err)
	}
	return nil
}

// DeleteSession removes a session row (logout).
func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	if err != nil {
		return fmt.Errorf("store: delete session: %w", err)
	}
	return nil
}

// PruneSessions deletes all expired session rows.
func (s *Store) PruneSessions() error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP`)
	if err != nil {
		return fmt.Errorf("store: prune sessions: %w", err)
	}
	return nil
}
