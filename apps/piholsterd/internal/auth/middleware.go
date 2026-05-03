package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/piholster/piholster/apps/piholsterd/internal/store"
)

const sessionCookieName = "piholster_session"

type contextKey int

const sessionContextKey contextKey = iota

// RequireAdmin is an HTTP middleware that validates the session cookie against
// the sessions table and refreshes last_seen on every request.
func RequireAdmin(st *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				writeUnauthorized(w, "session cookie missing")
				return
			}

			sess, err := st.GetSession(cookie.Value)
			if err != nil {
				slog.Error("session lookup failed", "err", err)
				writeUnauthorized(w, "session error")
				return
			}
			if sess == nil {
				writeUnauthorized(w, "session invalid or expired")
				return
			}

			if err := st.TouchSession(sess.Token); err != nil {
				// Non-fatal: log and continue. Touch failure must not block requests.
				slog.Warn("touch session failed", "err", err)
			}

			next.ServeHTTP(w, r.WithContext(
				contextWithSession(r.Context(), sess),
			))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
