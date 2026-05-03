package middleware

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

var defaultAllowed = []string{
	"piholster.local",
	"piholster.lan",
	"localhost",
	"127.0.0.1",
	"[::1]",
}

// AllowedHosts returns middleware that validates the Host header against a
// fixed allowlist plus any extra hosts provided by the caller or the
// EXTRA_ALLOWED_HOSTS environment variable (comma-separated).
// An unrecognised host causes a 421 Misdirected Request response.
func AllowedHosts(extra ...string) func(http.Handler) http.Handler {
	allowed := buildAllowset(extra)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := canonicalHost(r.Host)
			if _, ok := allowed[host]; ok {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMisdirectedRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "misdirected request: host not allowed",
			})
		})
	}
}

// buildAllowset merges the default list, caller-provided extras and the
// EXTRA_ALLOWED_HOSTS environment variable into a single lookup map.
func buildAllowset(extra []string) map[string]struct{} {
	set := make(map[string]struct{}, len(defaultAllowed)+len(extra)+4)

	for _, h := range defaultAllowed {
		set[h] = struct{}{}
	}
	for _, h := range extra {
		if h = strings.TrimSpace(h); h != "" {
			set[h] = struct{}{}
		}
	}

	if env := os.Getenv("EXTRA_ALLOWED_HOSTS"); env != "" {
		for _, h := range strings.Split(env, ",") {
			if h = strings.TrimSpace(h); h != "" {
				set[h] = struct{}{}
			}
		}
	}

	return set
}

// canonicalHost strips the port and lowercases the host component so that
// "PiHolster.Local:443" and "piholster.local" match the same allowlist entry.
func canonicalHost(raw string) string {
	host := raw
	// Strip port — handles IPv6 "[::1]:8443" as well.
	if last := strings.LastIndex(host, ":"); last != -1 {
		// Only strip if what follows looks like a port (all digits).
		maybePort := host[last+1:]
		allDigits := len(maybePort) > 0
		for _, c := range maybePort {
			if c < '0' || c > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			host = host[:last]
		}
	}
	return strings.ToLower(host)
}
