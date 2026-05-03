package alerts

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/piholster/piholster/apps/piholsterd/internal/arp"
)

// --- mock store ---

type mockStore struct {
	mu        sync.Mutex
	trusted   map[string]bool
	firstSeen map[string]time.Time
	upserted  []string
}

func newMockStore() *mockStore {
	return &mockStore{
		trusted:   make(map[string]bool),
		firstSeen: make(map[string]time.Time),
	}
}

func (m *mockStore) UpsertDevice(mac, ip, hostname string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upserted = append(m.upserted, mac)
	if _, exists := m.firstSeen[mac]; !exists {
		m.firstSeen[mac] = time.Now()
	}
	return nil
}

func (m *mockStore) IsDeviceTrusted(mac string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.trusted[mac], nil
}

func (m *mockStore) DeviceFirstSeen(mac string) (time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.firstSeen[mac]
	if !ok {
		return time.Time{}, nil
	}
	return t, nil
}

// --- fake Telegram server ---

type capturedRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type fakeServer struct {
	srv      *httptest.Server
	mu       sync.Mutex
	requests []capturedRequest
}

func newFakeServer(t *testing.T) *fakeServer {
	t.Helper()
	fs := &fakeServer{}
	fs.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var cr capturedRequest
		json.Unmarshal(body, &cr) //nolint:errcheck
		fs.mu.Lock()
		fs.requests = append(fs.requests, cr)
		fs.mu.Unlock()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
	}))
	t.Cleanup(fs.srv.Close)
	return fs
}

func (fs *fakeServer) captured() []capturedRequest {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	out := make([]capturedRequest, len(fs.requests))
	copy(out, fs.requests)
	return out
}

// newTestTelegramClient creates a TelegramClient that routes all requests
// through the fake server by overriding the HTTP transport.
func newTestTelegramClient(fs *fakeServer) *TelegramClient {
	c := NewTelegramClient("testtoken", "123456")
	c.http = &http.Client{
		Timeout:   5 * time.Second,
		Transport: &rewriteTransport{base: fs.srv.URL},
	}
	return c
}

// rewriteTransport replaces the scheme+host of every request with the test
// server's address so we never dial api.telegram.org in tests.
type rewriteTransport struct {
	base string
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	fakeURL := rt.base + req.URL.Path
	parsed, err := req.URL.Parse(fakeURL)
	if err != nil {
		return nil, err
	}
	cloned.URL = parsed
	cloned.Host = parsed.Host
	return http.DefaultTransport.RoundTrip(cloned)
}

// --- tests ---

// handle is called synchronously so there are no goroutine/timing issues.

func TestNotifierNewDevice(t *testing.T) {
	fs := newFakeServer(t)
	tg := newTestTelegramClient(fs)
	st := newMockStore()

	mac := "aa:bb:cc:dd:ee:ff"
	// Pre-seed first_seen to now so the device is "newly discovered".
	st.firstSeen[mac] = time.Now()

	n := NewNotifier(tg, st)
	n.handle(context.Background(), arp.Device{MAC: mac, IP: "192.168.1.42", Hostname: "iPhone"})

	reqs := fs.captured()
	if len(reqs) == 0 {
		t.Fatal("expected Telegram message for new untrusted device, got none")
	}
	msg := reqs[0]
	if msg.ChatID != "123456" {
		t.Errorf("chat_id: want 123456, got %q", msg.ChatID)
	}
	if msg.ParseMode != "HTML" {
		t.Errorf("parse_mode: want HTML, got %q", msg.ParseMode)
	}
	for _, want := range []string{"iPhone", "192.168.1.42", mac} {
		if !strContains(msg.Text, want) {
			t.Errorf("message text missing %q\ngot: %s", want, msg.Text)
		}
	}
}

func TestNotifierTrustedDevice(t *testing.T) {
	fs := newFakeServer(t)
	tg := newTestTelegramClient(fs)
	st := newMockStore()

	mac := "11:22:33:44:55:66"
	st.trusted[mac] = true
	st.firstSeen[mac] = time.Now()

	n := NewNotifier(tg, st)
	n.handle(context.Background(), arp.Device{MAC: mac, IP: "10.0.0.5", Hostname: "MyLaptop"})

	if reqs := fs.captured(); len(reqs) != 0 {
		t.Fatalf("expected no Telegram message for trusted device, got %d", len(reqs))
	}
}

func TestNotifierOldDevice(t *testing.T) {
	fs := newFakeServer(t)
	tg := newTestTelegramClient(fs)
	st := newMockStore()

	mac := "de:ad:be:ef:00:01"
	st.trusted[mac] = false
	st.firstSeen[mac] = time.Now().Add(-10 * time.Minute)

	n := NewNotifier(tg, st)
	n.handle(context.Background(), arp.Device{MAC: mac, IP: "10.0.0.9", Hostname: "OldDevice"})

	if reqs := fs.captured(); len(reqs) != 0 {
		t.Fatalf("expected no alert for device seen >5 min ago, got %d", len(reqs))
	}
}

func TestNotifierUnknownHostname(t *testing.T) {
	fs := newFakeServer(t)
	tg := newTestTelegramClient(fs)
	st := newMockStore()

	mac := "ca:fe:ba:be:00:01"
	st.firstSeen[mac] = time.Now()

	n := NewNotifier(tg, st)
	n.handle(context.Background(), arp.Device{MAC: mac, IP: "10.0.0.55", Hostname: ""})

	reqs := fs.captured()
	if len(reqs) == 0 {
		t.Fatal("expected Telegram message for device with no hostname")
	}
	if !strContains(reqs[0].Text, "Okänd enhet") {
		t.Errorf("expected fallback name 'Okänd enhet' in message, got: %s", reqs[0].Text)
	}
}

func TestTelegramClientSend(t *testing.T) {
	fs := newFakeServer(t)
	tg := newTestTelegramClient(fs)

	if err := tg.Send(context.Background(), "<b>Test</b>"); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	reqs := fs.captured()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	r := reqs[0]
	if r.ChatID != "123456" {
		t.Errorf("chat_id: want 123456, got %q", r.ChatID)
	}
	if r.Text != "<b>Test</b>" {
		t.Errorf("text: want <b>Test</b>, got %q", r.Text)
	}
	if r.ParseMode != "HTML" {
		t.Errorf("parse_mode: want HTML, got %q", r.ParseMode)
	}
}

func TestTelegramClientNoopWhenUnconfigured(t *testing.T) {
	fs := newFakeServer(t)
	tg := NewTelegramClient("", "")
	// Give the no-op client the same fake transport so any accidental request
	// would be caught.
	tg.http = &http.Client{Transport: &rewriteTransport{base: fs.srv.URL}}

	if err := tg.Send(context.Background(), "should not be sent"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqs := fs.captured(); len(reqs) != 0 {
		t.Fatalf("expected no HTTP call when token/chatID are empty, got %d", len(reqs))
	}
}

// strContains is strings.Contains reimplemented to avoid importing strings.
func strContains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
