package dns

import (
	"bufio"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// Blocklist holds domains that should be blocked, loaded from hosts-format files.
// The zero value is a usable empty blocklist.
type Blocklist struct {
	mu      sync.RWMutex
	entries map[string]struct{}
}

// NewBlocklist returns an initialised, empty Blocklist.
func NewBlocklist() *Blocklist {
	return &Blocklist{entries: make(map[string]struct{})}
}

// IsBlocked reports whether domain is on the blocklist.
// The lookup is case-insensitive and handles the trailing FQDN dot automatically.
func (b *Blocklist) IsBlocked(domain string) bool {
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	b.mu.RLock()
	_, ok := b.entries[domain]
	b.mu.RUnlock()
	return ok
}

// LoadFromFile reads a hosts-format file from disk and replaces the current entries.
func (b *Blocklist) LoadFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := b.LoadFromReader(f)
	if err != nil {
		return err
	}
	slog.Info("blocklist loaded", "path", path, "entries", n)
	return nil
}

// LoadFromReader parses hosts-format lines from r, merging into the existing entries.
// It returns the number of new domain rules added.
// Format: <addr> <domain> [<domain>...]
// Lines starting with '#' and blank lines are ignored.
func (b *Blocklist) LoadFromReader(r io.Reader) (int, error) {
	newEntries := make(map[string]struct{})
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		// hosts format: first field is address (0.0.0.0 or 127.0.0.1), rest are domains
		if len(fields) < 2 {
			continue
		}
		for _, domain := range fields[1:] {
			if strings.HasPrefix(domain, "#") {
				break
			}
			newEntries[strings.ToLower(domain)] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}

	b.mu.Lock()
	if b.entries == nil {
		b.entries = make(map[string]struct{})
	}
	for domain := range newEntries {
		b.entries[domain] = struct{}{}
	}
	b.mu.Unlock()

	return len(newEntries), nil
}

// Len returns the current number of blocked domains.
func (b *Blocklist) Len() int {
	b.mu.RLock()
	n := len(b.entries)
	b.mu.RUnlock()
	return n
}
