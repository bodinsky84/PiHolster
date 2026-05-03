package api

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/piholster/piholster/apps/piholsterd/internal/auth"
	"github.com/piholster/piholster/apps/piholsterd/internal/store"
)

const (
	sessionCookieName = "piholster_session"
	sessionTTL        = 30 * time.Minute
)

// AuthHandler groups the authentication HTTP handlers and their dependencies.
type AuthHandler struct {
	store   *store.Store
	limiter *auth.RateLimiter
}

func NewAuthHandler(st *store.Store, limiter *auth.RateLimiter) *AuthHandler {
	return &AuthHandler{store: st, limiter: limiter}
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)

	if !h.limiter.Allow(ip) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many attempts"})
		return
	}

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	user, err := h.store.GetUserByUsername(body.Username)
	if err != nil {
		slog.Error("login: store error", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Verify even when user is nil to prevent username enumeration via timing.
	var hashToCheck string
	if user != nil {
		hashToCheck = user.PasswordHash
	} else {
		// A constant dummy hash: burns Argon2id time regardless of outcome.
		hashToCheck = dummyHash
	}

	ok, err := auth.VerifyPassword(body.Password, hashToCheck)
	if err != nil {
		slog.Error("login: verify password", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if !ok || user == nil {
		h.limiter.Record(ip)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	token, err := auth.GenerateToken()
	if err != nil {
		slog.Error("login: generate token", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	expiresAt := time.Now().Add(sessionTTL)
	if err := h.store.CreateSession(token, user.ID, expiresAt); err != nil {
		slog.Error("login: create session", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if err := h.store.RecordLogin(user.ID); err != nil {
		slog.Warn("login: record login failed", "userID", user.ID, "err", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	writeJSON(w, http.StatusOK, map[string]bool{
		"must_change_password": user.MustChangePassword,
	})
}

// Logout handles POST /api/auth/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		if delErr := h.store.DeleteSession(cookie.Value); delErr != nil {
			slog.Warn("logout: delete session", "err", delErr)
		}
	}

	// Clear the cookie regardless of whether a session existed.
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	w.WriteHeader(http.StatusNoContent)
}

// ChangePassword handles POST /api/auth/change-password.
// Must be placed behind the RequireAdmin middleware.
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	sess := auth.SessionFromContext(r.Context())
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	var body struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if len(body.New) < 12 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "new password too short"})
		return
	}

	user, err := h.store.GetUserByID(sess.UserID)
	if err != nil || user == nil {
		slog.Error("change-password: get user", "userID", sess.UserID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	ok, err := auth.VerifyPassword(body.Current, user.PasswordHash)
	if err != nil {
		slog.Error("change-password: verify", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	newHash, err := auth.HashPassword(body.New)
	if err != nil {
		slog.Error("change-password: hash", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if err := h.store.UpdatePasswordHash(user.ID, newHash); err != nil {
		slog.Error("change-password: update hash", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if err := h.store.SetMustChangePassword(user.ID, false); err != nil {
		slog.Warn("change-password: clear must_change_password", "err", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// clientIP extracts the remote IP, stripping the port.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// dummyHash is a valid Argon2id hash for an impossible password used to
// normalise timing when the username is not found.
const dummyHash = "$argon2id$v=19$m=65536,t=3,p=2$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
