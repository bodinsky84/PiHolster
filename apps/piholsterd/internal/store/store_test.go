package store

import (
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenAndMigrate(t *testing.T) {
	s := openTestStore(t)

	tables := []string{"devices", "query_log", "settings", "users", "sessions"}
	for _, tbl := range tables {
		var name string
		err := s.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tbl,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found after migration: %v", tbl, err)
		}
	}
}

func TestUpsertAndListDevices(t *testing.T) {
	s := openTestStore(t)

	if err := s.UpsertDevice("aa:bb:cc:dd:ee:ff", "192.168.1.10", "my-laptop"); err != nil {
		t.Fatalf("UpsertDevice insert: %v", err)
	}

	// Second upsert with updated IP must not create a duplicate row.
	if err := s.UpsertDevice("aa:bb:cc:dd:ee:ff", "192.168.1.11", "my-laptop-renamed"); err != nil {
		t.Fatalf("UpsertDevice update: %v", err)
	}

	devices, err := s.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}

	d := devices[0]
	if d.IP != "192.168.1.11" {
		t.Errorf("expected updated IP, got %q", d.IP)
	}
	if d.Hostname != "my-laptop-renamed" {
		t.Errorf("expected updated hostname, got %q", d.Hostname)
	}
	if d.Trusted {
		t.Error("device should not be trusted by default")
	}

	if err := s.SetDeviceTrusted(d.MAC, true); err != nil {
		t.Fatalf("SetDeviceTrusted: %v", err)
	}
	if err := s.SetDeviceDisplayName(d.MAC, "Laptop"); err != nil {
		t.Fatalf("SetDeviceDisplayName: %v", err)
	}

	devices, err = s.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices after update: %v", err)
	}
	if !devices[0].Trusted {
		t.Error("device should be trusted after SetDeviceTrusted(true)")
	}
	if devices[0].DisplayName != "Laptop" {
		t.Errorf("unexpected display name: %q", devices[0].DisplayName)
	}
}

func TestLogQueryAndStats(t *testing.T) {
	s := openTestStore(t)

	before := time.Now().Add(-time.Second)

	if err := s.LogQuery("example.com", "10.0.0.1", false, "9.9.9.9", 5); err != nil {
		t.Fatalf("LogQuery: %v", err)
	}
	if err := s.LogQuery("ads.tracker.io", "10.0.0.2", true, "", 1); err != nil {
		t.Fatalf("LogQuery blocked: %v", err)
	}

	stats, err := s.QueryStats(before)
	if err != nil {
		t.Fatalf("QueryStats: %v", err)
	}
	if stats.Total != 2 {
		t.Errorf("expected 2 total, got %d", stats.Total)
	}
	if stats.Blocked != 1 {
		t.Errorf("expected 1 blocked, got %d", stats.Blocked)
	}

	// Pruning with a large window must not remove rows inserted seconds ago.
	if err := s.PruneOldLogs(7 * 24 * time.Hour); err != nil {
		t.Fatalf("PruneOldLogs: %v", err)
	}
	stats, err = s.QueryStats(before)
	if err != nil {
		t.Fatalf("QueryStats after prune: %v", err)
	}
	if stats.Total != 2 {
		t.Errorf("PruneOldLogs removed recent rows; expected 2, got %d", stats.Total)
	}
}

func TestSettingsGetSet(t *testing.T) {
	s := openTestStore(t)

	// Missing key must return an error.
	_, err := s.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing key, got nil")
	}

	fallback := s.GetOrDefault("nonexistent", "default-value")
	if fallback != "default-value" {
		t.Errorf("unexpected fallback: %q", fallback)
	}

	if err := s.Set("upstream", "9.9.9.9"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	v, err := s.Get("upstream")
	if err != nil {
		t.Fatalf("Get after Set: %v", err)
	}
	if v != "9.9.9.9" {
		t.Errorf("unexpected value: %q", v)
	}

	// Upsert an existing key.
	if err := s.Set("upstream", "1.1.1.1"); err != nil {
		t.Fatalf("Set update: %v", err)
	}
	v, err = s.Get("upstream")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if v != "1.1.1.1" {
		t.Errorf("Set did not update value: got %q", v)
	}
}
