package store

import (
	"database/sql"
	"errors"
	"time"
)

type Device struct {
	ID          int64
	MAC         string
	IP          string
	Hostname    string
	DisplayName string
	Trusted     bool
	FirstSeen   time.Time
	LastSeen    time.Time
}

// UpsertDevice inserts a new device or updates ip, hostname and last_seen for
// an existing one, identified by mac. first_seen is intentionally absent from
// the UPDATE clause so it retains the original discovery timestamp forever.
func (s *Store) UpsertDevice(mac, ip, hostname string) error {
	_, err := s.db.Exec(`
		INSERT INTO devices (mac, ip, hostname, last_seen)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(mac) DO UPDATE SET
			ip        = excluded.ip,
			hostname  = excluded.hostname,
			last_seen = CURRENT_TIMESTAMP
	`, mac, ip, hostname)
	return err
}

// IsDeviceTrusted reports whether the device with the given MAC address has
// been marked as trusted. Returns false and no error when the device is unknown.
func (s *Store) IsDeviceTrusted(mac string) (bool, error) {
	var trusted int
	err := s.db.QueryRow(
		`SELECT trusted FROM devices WHERE mac = ?`, mac,
	).Scan(&trusted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return trusted != 0, nil
}

// DeviceFirstSeen returns the first_seen timestamp for the given MAC.
// Returns the zero time and no error when the device is unknown.
func (s *Store) DeviceFirstSeen(mac string) (time.Time, error) {
	var raw string
	err := s.db.QueryRow(
		`SELECT first_seen FROM devices WHERE mac = ?`, mac,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	t, _ := parseSQLiteTime(raw)
	return t, nil
}

func (s *Store) ListDevices() ([]Device, error) {
	rows, err := s.db.Query(`
		SELECT id, mac, ip, hostname, display_name, trusted, first_seen, last_seen
		FROM devices
		ORDER BY last_seen DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		var trusted int
		// SQLite stores DATETIME as TEXT; scan into strings and parse manually.
		var firstSeen, lastSeen string
		if err := rows.Scan(
			&d.ID, &d.MAC, &d.IP, &d.Hostname, &d.DisplayName,
			&trusted, &firstSeen, &lastSeen,
		); err != nil {
			return nil, err
		}
		d.Trusted = trusted != 0
		d.FirstSeen, _ = parseSQLiteTime(firstSeen)
		d.LastSeen, _ = parseSQLiteTime(lastSeen)
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (s *Store) SetDeviceTrusted(mac string, trusted bool) error {
	t := 0
	if trusted {
		t = 1
	}
	_, err := s.db.Exec(
		`UPDATE devices SET trusted = ? WHERE mac = ?`,
		t, mac,
	)
	return err
}

func (s *Store) SetDeviceDisplayName(mac, name string) error {
	_, err := s.db.Exec(
		`UPDATE devices SET display_name = ? WHERE mac = ?`,
		name, mac,
	)
	return err
}

// parseSQLiteTime handles the two formats SQLite uses for CURRENT_TIMESTAMP.
func parseSQLiteTime(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, nil
}
