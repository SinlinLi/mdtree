package auth

import (
	"sync"
	"time"
)

// RateLimiter throttles repeated failures per key (typically a client IP)
// using a fixed-window counter. It slows down password brute-force attempts.
type RateLimiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	entries map[string]*rlEntry
}

type rlEntry struct {
	count       int
	windowStart time.Time
}

// NewRateLimiter allows at most max failures per key within window.
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{max: max, window: window, entries: make(map[string]*rlEntry)}
}

// Allowed reports whether key may make another attempt.
func (r *RateLimiter) Allowed(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.current(key).count < r.max
}

// Fail records a failed attempt for key.
func (r *RateLimiter) Fail(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.current(key).count++
}

// Reset clears the failure counter for key after a successful attempt.
func (r *RateLimiter) Reset(key string) {
	r.mu.Lock()
	delete(r.entries, key)
	r.mu.Unlock()
}

// RetryAfter returns how long key must wait before its window resets.
func (r *RateLimiter) RetryAfter(key string) time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[key]
	if !ok {
		return 0
	}
	remaining := r.window - time.Since(e.windowStart)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// current returns the live entry for key, resetting it if the window elapsed.
// The caller must hold r.mu.
func (r *RateLimiter) current(key string) *rlEntry {
	now := time.Now()
	e, ok := r.entries[key]
	if !ok || now.Sub(e.windowStart) > r.window {
		e = &rlEntry{windowStart: now}
		r.entries[key] = e
	}
	return e
}
