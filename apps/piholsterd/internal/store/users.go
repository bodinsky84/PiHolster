package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// User represents a row in the users table.
type User struct {
	ID                 int64
	Username           string
	PasswordHash       string
	MustChangePassword bool
	CreatedAt          time.Time
	LastLogin          *time.Time
}

// CreateUser inserts a new user. The caller must supply a pre-hashed password.
func (s *Store) CreateUser(username, passwordHash string) error {
	_, err := s.db.Exec(
		`INSERT INTO users (username, password_hash, must_change_password)
		 VALUES (?, ?, 1)`,
		username, passwordHash,
	)
	if err != nil {
		return fmt.Errorf("store: create user %q: %w", username, err)
	}
	return nil
}

// GetUserByUsername returns the user or (nil, nil) when the username does not
// exist. Any other error is returned as-is.
func (s *Store) GetUserByUsername(username string) (*User, error) {
	u := &User{}
	var mustChange int
	var lastLogin sql.NullTime

	err := s.db.QueryRow(
		`SELECT id, username, password_hash, must_change_password, created_at, last_login
		 FROM users WHERE username = ?`,
		username,
	).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &mustChange, &u.CreatedAt, &lastLogin,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get user %q: %w", username, err)
	}

	u.MustChangePassword = mustChange != 0
	if lastLogin.Valid {
		t := lastLogin.Time
		u.LastLogin = &t
	}
	return u, nil
}

// GetUserByID returns the user or (nil, nil) when the ID does not exist.
func (s *Store) GetUserByID(id int64) (*User, error) {
	u := &User{}
	var mustChange int
	var lastLogin sql.NullTime

	err := s.db.QueryRow(
		`SELECT id, username, password_hash, must_change_password, created_at, last_login
		 FROM users WHERE id = ?`,
		id,
	).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &mustChange, &u.CreatedAt, &lastLogin,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get user by id %d: %w", id, err)
	}

	u.MustChangePassword = mustChange != 0
	if lastLogin.Valid {
		t := lastLogin.Time
		u.LastLogin = &t
	}
	return u, nil
}

// UserCount returns the number of rows in the users table.
func (s *Store) UserCount() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("store: count users: %w", err)
	}
	return n, nil
}

func (s *Store) SetMustChangePassword(userID int64, must bool) error {
	val := 0
	if must {
		val = 1
	}
	_, err := s.db.Exec(
		`UPDATE users SET must_change_password = ? WHERE id = ?`,
		val, userID,
	)
	if err != nil {
		return fmt.Errorf("store: set must_change_password user %d: %w", userID, err)
	}
	return nil
}

func (s *Store) UpdatePasswordHash(userID int64, hash string) error {
	_, err := s.db.Exec(
		`UPDATE users SET password_hash = ? WHERE id = ?`,
		hash, userID,
	)
	if err != nil {
		return fmt.Errorf("store: update password hash user %d: %w", userID, err)
	}
	return nil
}

func (s *Store) RecordLogin(userID int64) error {
	_, err := s.db.Exec(
		`UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = ?`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("store: record login user %d: %w", userID, err)
	}
	return nil
}
