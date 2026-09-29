package storage

import (
	"sync"
	"time"
)

// rateLimiter is a fixed-window counter per key (client IP). It lives in
// memory, so counts reset when the server restarts; that is fine for
// keeping a single client from flooding the store.
type rateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string]*windowCount
}

type windowCount struct {
	start time.Time
	count int
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	if window <= 0 {
		window = time.Hour
	}
	return &rateLimiter{limit: limit, window: window, hits: map[string]*windowCount{}}
}

func (r *rateLimiter) Allow(key string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	w, ok := r.hits[key]
	if !ok || now.Sub(w.start) >= r.window {
		r.hits[key] = &windowCount{start: now, count: 1}
		return true
	}
	if w.count >= r.limit {
		return false
	}
	w.count++
	return true
}

// Prune drops windows that have ended so the map doesn't grow without bound.
func (r *rateLimiter) Prune(now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for key, w := range r.hits {
		if now.Sub(w.start) >= r.window {
			delete(r.hits, key)
		}
	}
}
