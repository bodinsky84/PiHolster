package smoke

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
)

// piIP is the target Pi's IP, read once at TestMain.
var piIP string

// smokeTimeout is the per-test HTTP client timeout.
var smokeTimeout time.Duration

// bootTimeout is the max wait for TestBootTime.
var bootTimeout time.Duration

// httpClient is shared across all tests with InsecureSkipVerify for self-signed certs.
var httpClient *http.Client

func TestMain(m *testing.M) {
	piIP = os.Getenv("PI_IP")
	if piIP == "" {
		fmt.Fprintln(os.Stderr, "PI_IP not set — skipping smoke tests")
		os.Exit(0)
	}

	smokeTimeout = parseDuration("SMOKE_TIMEOUT", 30*time.Second)
	bootTimeout = parseDuration("BOOT_TIMEOUT", 90*time.Second)

	httpClient = &http.Client{
		Timeout: smokeTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-signed cert on Pi
		},
	}

	os.Exit(m.Run())
}

// TestBootTime polls /api/health until 200 OK, measuring elapsed time.
// Fails if the Pi does not become responsive within bootTimeout.
func TestBootTime(t *testing.T) {
	url := fmt.Sprintf("https://%s/api/health", piIP)
	deadline := time.Now().Add(bootTimeout)
	start := time.Now()

	for time.Now().Before(deadline) {
		resp, err := httpClient.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				elapsed := time.Since(start)
				t.Logf("Pi became healthy in %s", elapsed)
				return
			}
			t.Logf("got HTTP %d, continuing to poll", resp.StatusCode)
		} else {
			t.Logf("poll error: %v", err)
		}
		time.Sleep(2 * time.Second)
	}

	t.Fatalf("Pi did not respond with 200 OK within %s", bootTimeout)
}

// TestFirewallPreFirstboot verifies port 80 is reachable (or not) depending on
// the firewall state. Set SKIP_FIREWALL_TEST=1 to skip when boot timing is
// unpredictable.
//
// The test expects a connection to succeed (the web UI is up), but checks that
// a raw TCP connect to :80 completes within 5 s — a firewall that silently
// drops packets would cause a timeout rather than connection refused, which is
// the failure mode we guard against.
func TestFirewallPreFirstboot(t *testing.T) {
	if os.Getenv("SKIP_FIREWALL_TEST") == "1" {
		t.Skip("SKIP_FIREWALL_TEST=1")
	}

	addr := fmt.Sprintf("%s:80", piIP)
	// We use a 5 s dial — if the firewall drops packets we get a timeout, which
	// means the firewall is incorrectly blocking traffic we expect to reach the UI.
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		// connection refused is fine — port is closed by choice.
		// A timeout is the problematic case (silent drop).
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			t.Fatalf("TCP connect to %s timed out — firewall may be silently dropping packets", addr)
		}
		t.Logf("port 80 refused (closed): %v", err)
		return
	}
	conn.Close()
	t.Logf("port 80 is open and accepting connections")
}

// TestHTTPSRedirect verifies that port 80 redirects to https:// (US-24).
// The test uses a client that does NOT follow redirects so we can inspect the
// Location header directly.
func TestHTTPSRedirect(t *testing.T) {
	noRedirectClient := &http.Client{
		Timeout: smokeTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-signed cert on Pi
		},
	}

	url := fmt.Sprintf("http://%s/", piIP)
	resp, err := noRedirectClient.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("expected 301 Moved Permanently, got %d", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	if loc == "" {
		t.Fatal("Location header missing in redirect response")
	}
	if !strings.HasPrefix(loc, "https://") {
		t.Fatalf("Location %q does not start with https://", loc)
	}
	t.Logf("HTTP→HTTPS redirect: %s → %s", url, loc)
}

// TestDNSLatency sends 20 DNS queries to PI_IP:53 (mix of blocked and
// legitimate domains) and fails if the median round-trip exceeds 20 ms.
func TestDNSLatency(t *testing.T) {
	queries := []string{
		// likely blocked by default Pi-hole style lists
		"doubleclick.net.",
		"googlesyndication.com.",
		"adnxs.com.",
		"ads.example.com.",
		"tracking.example.net.",
		// legitimate domains that should be forwarded upstream
		"github.com.",
		"cloudflare.com.",
		"google.com.",
		"dns.google.",
		"example.com.",
		// repeat to reach 20 queries
		"doubleclick.net.",
		"github.com.",
		"adnxs.com.",
		"cloudflare.com.",
		"googlesyndication.com.",
		"google.com.",
		"ads.example.com.",
		"dns.google.",
		"tracking.example.net.",
		"example.com.",
	}

	server := fmt.Sprintf("%s:53", piIP)
	c := new(dns.Client)
	c.Timeout = smokeTimeout

	latencies := make([]float64, 0, len(queries))

	for _, q := range queries {
		m := new(dns.Msg)
		m.SetQuestion(q, dns.TypeA)
		m.RecursionDesired = true

		start := time.Now()
		_, _, err := c.Exchange(m, server)
		elapsed := time.Since(start)

		if err != nil {
			t.Logf("DNS query %s error: %v", q, err)
			continue
		}
		ms := float64(elapsed.Nanoseconds()) / 1e6
		t.Logf("DNS %s: %.2f ms", q, ms)
		latencies = append(latencies, ms)
	}

	if len(latencies) == 0 {
		t.Fatal("all DNS queries failed — is :53 reachable?")
	}

	med := median(latencies)
	t.Logf("DNS median latency: %.2f ms over %d queries", med, len(latencies))

	const maxMedianMs = 20.0
	if med > maxMedianMs {
		t.Fatalf("DNS median latency %.2f ms exceeds threshold of %.0f ms", med, maxMedianMs)
	}
}

// TestRAMUsage checks the optional ram_used_mb field in /api/health.
// If the field is absent the test is skipped — the daemon may not expose it yet.
func TestRAMUsage(t *testing.T) {
	url := fmt.Sprintf("https://%s/api/health", piIP)

	resp, err := httpClient.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("JSON parse: %v — body: %s", err, body)
	}

	raw, present := payload["ram_used_mb"]
	if !present {
		t.Skip("ram_used_mb not present in /api/health — field not yet implemented")
	}

	var ramMB float64
	if err := json.Unmarshal(raw, &ramMB); err != nil {
		t.Fatalf("ram_used_mb is not a number: %v", err)
	}

	t.Logf("RAM used: %.1f MB", ramMB)

	const maxRAMMB = 300.0
	if ramMB > maxRAMMB {
		t.Fatalf("RAM usage %.1f MB exceeds threshold of %.0f MB", ramMB, maxRAMMB)
	}
}

// TestWebUIResponds verifies that the root path returns 200 with an HTML body.
func TestWebUIResponds(t *testing.T) {
	url := fmt.Sprintf("https://%s/", piIP)

	resp, err := httpClient.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}

	if !strings.Contains(strings.ToLower(string(body)), "<html") {
		t.Fatalf("response body does not contain <html — got: %.200s", body)
	}
	t.Logf("web UI responded with %d bytes of HTML", len(body))
}

// TestAPIHealth verifies /api/health returns 200 and a JSON body that
// contains either "status":"ok" or "dns_ok":true.
func TestAPIHealth(t *testing.T) {
	url := fmt.Sprintf("https://%s/api/health", piIP)

	resp, err := httpClient.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}

	t.Logf("health response: %s", body)

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("JSON parse: %v — body: %s", err, body)
	}

	statusOK := false
	if raw, ok := payload["status"]; ok {
		var s string
		if json.Unmarshal(raw, &s) == nil && s == "ok" {
			statusOK = true
		}
	}

	dnsOK := false
	if raw, ok := payload["dns_ok"]; ok {
		var b bool
		if json.Unmarshal(raw, &b) == nil && b {
			dnsOK = true
		}
	}
	// dns_running is the field actually present in piholsterd — accept it too.
	if raw, ok := payload["dns_running"]; ok {
		var b bool
		if json.Unmarshal(raw, &b) == nil && b {
			dnsOK = true
		}
	}

	if !statusOK && !dnsOK {
		t.Fatalf(`health response missing "status":"ok" or "dns_ok":true — got: %s`, body)
	}
}

// median returns the middle value of a sorted copy of vs.
// Returns 0 for an empty slice.
func median(vs []float64) float64 {
	if len(vs) == 0 {
		return 0
	}
	cp := make([]float64, len(vs))
	copy(cp, vs)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 0 {
		return (cp[n/2-1] + cp[n/2]) / 2
	}
	return cp[n/2]
}

// parseDuration reads a duration from an env variable. Falls back to def.
func parseDuration(env string, def time.Duration) time.Duration {
	v := os.Getenv(env)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: invalid %s=%q, using default %s\n", env, v, def)
		return def
	}
	return d
}

