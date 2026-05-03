package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/piholster/piholster/apps/piholsterd/internal/api/middleware"
)

// nopHandler is a minimal handler that records that it was called.
var nopHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func makeRequest(handler http.Handler, host string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if host != "" {
		req.Host = host
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// --- SecurityHeaders tests ---

func TestSecurityHeadersPresent(t *testing.T) {
	t.Parallel()

	want := []string{
		"Content-Security-Policy",
		"Strict-Transport-Security",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"Referrer-Policy",
		"Permissions-Policy",
		"Cross-Origin-Embedder-Policy",
		"Cross-Origin-Opener-Policy",
	}

	handler := middleware.SecurityHeaders(nopHandler)
	rec := makeRequest(handler, "")

	for _, header := range want {
		if got := rec.Header().Get(header); got == "" {
			t.Errorf("missing header %q", header)
		}
	}
}

func TestCSPIsStrictSelf(t *testing.T) {
	t.Parallel()

	handler := middleware.SecurityHeaders(nopHandler)
	rec := makeRequest(handler, "")
	csp := rec.Header().Get("Content-Security-Policy")

	for _, want := range []string{"script-src 'self'", "style-src 'self'"} {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP %q does not contain %q", csp, want)
		}
	}
	for _, bad := range []string{"'unsafe-inline'", "nonce-"} {
		if strings.Contains(csp, bad) {
			t.Errorf("CSP %q must not contain %q", csp, bad)
		}
	}
}

func TestCSPIsIdempotent(t *testing.T) {
	t.Parallel()

	handler := middleware.SecurityHeaders(nopHandler)
	csp1 := makeRequest(handler, "").Header().Get("Content-Security-Policy")
	csp2 := makeRequest(handler, "").Header().Get("Content-Security-Policy")

	if csp1 != csp2 {
		t.Errorf("CSP must be identical across requests; got %q vs %q", csp1, csp2)
	}
}

// --- AllowedHosts tests ---

func TestAllowedHostsBlocks(t *testing.T) {
	t.Parallel()

	handler := middleware.AllowedHosts()(nopHandler)
	rec := makeRequest(handler, "evil.com")

	if rec.Code != http.StatusMisdirectedRequest {
		t.Errorf("expected 421, got %d", rec.Code)
	}
}

func TestAllowedHostsAllows(t *testing.T) {
	t.Parallel()

	cases := []string{
		"piholster.local",
		"piholster.lan",
		"localhost",
		"127.0.0.1",
		"[::1]",
		"piholster.local:443",
		"localhost:8080",
	}

	handler := middleware.AllowedHosts()(nopHandler)

	for _, host := range cases {
		host := host
		t.Run(host, func(t *testing.T) {
			t.Parallel()
			rec := makeRequest(handler, host)
			if rec.Code != http.StatusOK {
				t.Errorf("host %q: expected 200, got %d", host, rec.Code)
			}
		})
	}
}

func TestAllowedHostsExtra(t *testing.T) {
	t.Parallel()

	handler := middleware.AllowedHosts("custom.internal")(nopHandler)

	rec := makeRequest(handler, "custom.internal")
	if rec.Code != http.StatusOK {
		t.Errorf("extra host: expected 200, got %d", rec.Code)
	}

	rec2 := makeRequest(handler, "not-allowed.example")
	if rec2.Code != http.StatusMisdirectedRequest {
		t.Errorf("unknown host: expected 421, got %d", rec2.Code)
	}
}

func TestAllowedHostsEnvVar(t *testing.T) {
	// Not parallel — modifies environment.
	t.Setenv("EXTRA_ALLOWED_HOSTS", "env-host.internal,another.internal")

	handler := middleware.AllowedHosts()(nopHandler)

	for _, host := range []string{"env-host.internal", "another.internal"} {
		rec := makeRequest(handler, host)
		if rec.Code != http.StatusOK {
			t.Errorf("env host %q: expected 200, got %d", host, rec.Code)
		}
	}
}

// --- Chain tests ---

func TestChainIdentity(t *testing.T) {
	t.Parallel()

	// Chain with no middleware must behave identically to the bare handler.
	handler := middleware.Chain()(nopHandler)
	rec := makeRequest(handler, "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestChainOrder(t *testing.T) {
	t.Parallel()

	// Each middleware appends its label to the X-Trace header so we can verify
	// that the outermost middleware runs first (declaration order).
	label := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				prev := w.Header().Get("X-Trace")
				if prev == "" {
					w.Header().Set("X-Trace", name)
				} else {
					w.Header().Set("X-Trace", prev+","+name)
				}
				next.ServeHTTP(w, r)
			})
		}
	}

	handler := middleware.Chain(label("a"), label("b"), label("c"))(nopHandler)
	rec := makeRequest(handler, "")

	if got := rec.Header().Get("X-Trace"); got != "a,b,c" {
		t.Errorf("middleware execution order: want %q, got %q", "a,b,c", got)
	}
}

func TestChainComposesAllowedHostsAndSecurityHeaders(t *testing.T) {
	t.Parallel()

	handler := middleware.Chain(
		middleware.AllowedHosts(),
		middleware.SecurityHeaders,
	)(nopHandler)

	// Known host must pass through and receive security headers.
	rec := makeRequest(handler, "piholster.local")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for known host, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Security-Policy"); got == "" {
		t.Error("CSP header missing after Chain")
	}

	// Unknown host must be rejected before SecurityHeaders even runs.
	rec2 := makeRequest(handler, "evil.com")
	if rec2.Code != http.StatusMisdirectedRequest {
		t.Errorf("expected 421 for unknown host, got %d", rec2.Code)
	}
}
