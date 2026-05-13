package api

import (
	"context"
	"net/http"

	"github.com/piholster/piholster/apps/piholsterd/internal/allsvenskan"
	"github.com/piholster/piholster/apps/piholsterd/internal/api/middleware"
	"github.com/piholster/piholster/apps/piholsterd/internal/auth"
	"github.com/piholster/piholster/apps/piholsterd/internal/store"
)

// NewRouter builds and returns the HTTP handler with the full middleware stack
// wrapping all API routes.
//
// Middleware order (ADR-002 §3.3.2):
//  1. AllowedHosts — DNS-rebinding protection, returns 421 for unknown hosts.
//  2. SecurityHeaders — per-request CSP nonce + all security response headers.
//
// Per-route auth is applied via requireAdmin inside registerRoutes.
// Requires Go 1.22+ for path-parameter syntax in HandleFunc patterns.
func NewRouter(ctx context.Context, st *store.Store, asEngine *allsvenskan.Engine) http.Handler {
	mux := http.NewServeMux()

	limiter := auth.NewRateLimiter(ctx)

	healthHandler := NewHealthHandler(st)
	statusHandler := NewStatusHandler(st)
	devicesHandler := NewDevicesHandler(st)
	authHandler := NewAuthHandler(st, limiter)
	allsvenskanHandler := NewAllsvenskanHandler(asEngine)

	requireAdmin := auth.RequireAdmin(st)

	mux.HandleFunc("GET /api/health", healthHandler.Health)

	mux.Handle("GET /api/status",
		requireAdmin(http.HandlerFunc(statusHandler.Status)),
	)
	mux.Handle("GET /api/devices",
		requireAdmin(http.HandlerFunc(devicesHandler.List)),
	)

	mux.Handle("GET /api/allsvenskan/table",
		requireAdmin(http.HandlerFunc(allsvenskanHandler.Table)),
	)
	mux.Handle("GET /api/allsvenskan/news",
		requireAdmin(http.HandlerFunc(allsvenskanHandler.News)),
	)
	mux.Handle("GET /api/allsvenskan/matches",
		requireAdmin(http.HandlerFunc(allsvenskanHandler.Matches)),
	)
	mux.Handle("GET /api/allsvenskan/stats",
		requireAdmin(http.HandlerFunc(allsvenskanHandler.Stats)),
	)

	mux.Handle("POST /api/devices/{mac}/trust",
		requireAdmin(http.HandlerFunc(devicesHandler.SetTrust)),
	)
	mux.Handle("POST /api/devices/{mac}/rename",
		requireAdmin(http.HandlerFunc(devicesHandler.Rename)),
	)

	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)

	mux.Handle("POST /api/auth/change-password",
		requireAdmin(http.HandlerFunc(authHandler.ChangePassword)),
	)

	// Catch-all: serve the embedded SvelteKit build. Must be registered last
	// so API routes take precedence.
	mux.Handle("/", StaticHandler())

	return middleware.AllowedHosts()(
		middleware.SecurityHeaders(mux),
	)
}
