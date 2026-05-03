package auth

import (
	"context"

	"github.com/piholster/piholster/apps/piholsterd/internal/store"
)

func contextWithSession(ctx context.Context, sess *store.Session) context.Context {
	return context.WithValue(ctx, sessionContextKey, sess)
}

// SessionFromContext retrieves the validated session injected by RequireAdmin.
// Returns nil when no session is present (e.g. unauthenticated routes).
func SessionFromContext(ctx context.Context) *store.Session {
	sess, _ := ctx.Value(sessionContextKey).(*store.Session)
	return sess
}
