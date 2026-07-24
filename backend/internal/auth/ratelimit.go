package auth

import (
	"sync"
	"time"
)

// LoginLimiter is a minimal in-memory fixed-window rate limiter. State is
// per-process and does not survive a restart or scale across instances —
// acceptable for a single-process MVP.
type LoginLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	max      int
	window   time.Duration
}

func NewLoginLimiter(max int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{
		attempts: make(map[string][]time.Time),
		max:      max,
		window:   window,
	}
}

// Allow records an attempt for key and reports whether it's within the
// allowed rate. Call it once per login attempt, regardless of outcome.
func (l *LoginLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	kept := l.attempts[key][:0]
	for _, t := range l.attempts[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}

	if len(kept) >= l.max {
		l.attempts[key] = kept
		return false
	}

	l.attempts[key] = append(kept, now)
	return true
}
