package dns_test

import (
	"strings"
	"testing"

	internaldns "github.com/piholster/piholster/apps/piholsterd/internal/dns"
)

const sampleHosts = `
# PiHolster test blocklist
0.0.0.0 doubleclick.net
0.0.0.0 googlesyndication.com
0.0.0.0 adnxs.com
127.0.0.1 ads.example.com  # inline comment
0.0.0.0 UPPERCASE.EXAMPLE.COM
`

func TestLoadFromReader(t *testing.T) {
	bl := internaldns.NewBlocklist()
	n, err := bl.LoadFromReader(strings.NewReader(sampleHosts))
	if err != nil {
		t.Fatalf("LoadFromReader returned unexpected error: %v", err)
	}
	// sampleHosts contains 5 unique domain rules
	if n != 5 {
		t.Errorf("expected 5 entries loaded, got %d", n)
	}
	if bl.Len() != 5 {
		t.Errorf("expected Len() == 5, got %d", bl.Len())
	}
}

func TestIsBlocked(t *testing.T) {
	bl := internaldns.NewBlocklist()
	if _, err := bl.LoadFromReader(strings.NewReader(sampleHosts)); err != nil {
		t.Fatalf("LoadFromReader: %v", err)
	}

	cases := []struct {
		domain  string
		blocked bool
	}{
		{"doubleclick.net", true},
		{"doubleclick.net.", true},   // FQDN trailing dot
		{"DOUBLECLICK.NET", true},    // case-insensitive
		{"googlesyndication.com", true},
		{"adnxs.com", true},
		{"ads.example.com", true},
		{"uppercase.example.com", true}, // was loaded in uppercase, must normalise
	}

	for _, tc := range cases {
		t.Run(tc.domain, func(t *testing.T) {
			got := bl.IsBlocked(tc.domain)
			if got != tc.blocked {
				t.Errorf("IsBlocked(%q) = %v, want %v", tc.domain, got, tc.blocked)
			}
		})
	}
}

func TestIsNotBlocked(t *testing.T) {
	bl := internaldns.NewBlocklist()
	if _, err := bl.LoadFromReader(strings.NewReader(sampleHosts)); err != nil {
		t.Fatalf("LoadFromReader: %v", err)
	}

	safe := []string{
		"google.com",
		"example.com",        // parent of ads.example.com — must NOT be blocked
		"sub.doubleclick.net", // subdomain — must NOT be blocked (exact match only)
		"cloudflare.com",
		"quad9.net",
		"piholster.local",
		"",
	}

	for _, domain := range safe {
		if bl.IsBlocked(domain) {
			t.Errorf("IsBlocked(%q) = true, want false (should not be blocked)", domain)
		}
	}
}

func TestLoadFromReaderEmptyInput(t *testing.T) {
	bl := internaldns.NewBlocklist()
	n, err := bl.LoadFromReader(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error on empty input: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 entries from empty input, got %d", n)
	}
}

func TestLoadFromReaderCommentsAndBlanks(t *testing.T) {
	input := `
# comment

  # another comment
`
	bl := internaldns.NewBlocklist()
	n, err := bl.LoadFromReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 entries, got %d", n)
	}
}

func TestLoadFromReaderMerges(t *testing.T) {
	bl := internaldns.NewBlocklist()
	_, _ = bl.LoadFromReader(strings.NewReader("0.0.0.0 first.example.com\n"))
	_, _ = bl.LoadFromReader(strings.NewReader("0.0.0.0 second.example.com\n"))

	if !bl.IsBlocked("first.example.com") {
		t.Error("first.example.com should still be blocked after second load")
	}
	if !bl.IsBlocked("second.example.com") {
		t.Error("second.example.com should be blocked after second load")
	}
}
