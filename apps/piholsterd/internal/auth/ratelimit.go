package auth

import (
	"context"
	"sync"
	"time"
)

const (
	maxAttempts = 5
	lockoutDur  = 60 * time.Second
	pruneEvery  = 5 * time.Minute
)

// RateLimiter tracks failed login attempts per source IP using an in-memory
// sliding window. Safe for concurrent use.
type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

// NewRateLimiter creates a RateLimiter whose background prune goroutine exits
// when ctx is cancelled, preventing goroutine leaks in tests and on shutdown.
func NewRateLimiter(ctx context.Context) *RateLimiter {
	r := &RateLimiter{
		attempts: make(map[string][]time.Time),
	}
	go r.pruneLoop(ctx)
	return r
}

// Allow returns false when the IP has exceeded maxAttempts within lockoutDur.
func (r *RateLimiter) Allow(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-lockoutDur)
	recent := filterRecent(r.attempts[ip], cutoff)
	r.attempts[ip] = recent
	return len(recent) < maxAttempts
}

// Record registers a failed attempt for ip.
func (r *RateLimiter) Record(ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.attempts[ip] = append(r.attempts[ip], time.Now())
}

func (r *RateLimiter) pruneLoop(ctx context.Context) {
	ticker := time.NewTicker(pruneEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-lockoutDur)
			r.mu.Lock()
			for ip, times := range r.attempts {
				recent := filterRecent(times, cutoff)
				if len(recent) == 0 {
					delete(r.attempts, ip)
				} else {
					r.attempts[ip] = recent
				}
			}
			r.mu.Unlock()
		}
	}
}

func filterRecent(times []time.Time, cutoff time.Time) []time.Time {
	// Timestamps are appended in order, so the oldest are at the front.
	i := 0
	for i < len(times) && times[i].Before(cutoff) {
		i++
	}
	return times[i:]
}
