// Package throttle implements a small in-process sliding-window limiter for
// authentication and resend throttling. Single-instance deployments are the
// current target; a shared store is required if the API is horizontally scaled.
package throttle

import (
	"sync"
	"time"
)

// Limiter tracks attempts per key within a rolling window.
type Limiter struct {
	mu          sync.Mutex
	window      time.Duration
	max         int
	entries     map[string][]time.Time
	lastCleanup time.Time
	now         func() time.Time
}

func New(window time.Duration, max int) *Limiter {
	if window <= 0 {
		window = time.Minute
	}
	if max <= 0 {
		max = 10
	}
	return &Limiter{window: window, max: max, entries: make(map[string][]time.Time), now: time.Now}
}

// Allow records an attempt and reports whether it is within the limit.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if l.lastCleanup.IsZero() || now.Sub(l.lastCleanup) >= l.window {
		l.cleanupLocked(now)
		l.lastCleanup = now
	}
	cutoff := now.Add(-l.window)
	attempts := l.entries[key]
	kept := attempts[:0]
	for _, at := range attempts {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	if len(kept) >= l.max {
		l.entries[key] = kept
		return false
	}
	l.entries[key] = append(kept, now)
	return true
}

// Reset clears attempts for a key after a successful authentication.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

// Cleanup removes expired entries; call periodically to bound memory.
func (l *Limiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.cleanupLocked(now)
	l.lastCleanup = now
}

func (l *Limiter) cleanupLocked(now time.Time) {
	cutoff := now.Add(-l.window)
	for key, attempts := range l.entries {
		kept := attempts[:0]
		for _, at := range attempts {
			if at.After(cutoff) {
				kept = append(kept, at)
			}
		}
		if len(kept) == 0 {
			delete(l.entries, key)
		} else {
			l.entries[key] = kept
		}
	}
}
