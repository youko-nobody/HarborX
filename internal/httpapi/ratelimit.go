package httpapi

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// loginRateLimiter is a small in-memory sliding-window limiter used to slow
// down brute-force attempts against the login endpoint. It is not a global
// rate limiter and makes no durability guarantees; it exists to raise the
// cost of guessing credentials from a single host.
type loginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	max      int
	window   time.Duration
	cleans   int
}

func newLoginRateLimiter(max int, window time.Duration) *loginRateLimiter {
	if max <= 0 {
		max = 5
	}
	if window <= 0 {
		window = time.Minute
	}
	return &loginRateLimiter{
		attempts: make(map[string][]time.Time),
		max:      max,
		window:   window,
	}
}

// allow reports whether a new attempt from key is permitted. Every call
// records an attempt for the key; callers should only invoke allow for
// attempts they actually process.
func (l *loginRateLimiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-l.window)
	recent := l.attempts[key][:0]
	for _, t := range l.attempts[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	l.attempts[key] = recent

	if len(recent) >= l.max {
		return false
	}
	l.attempts[key] = append(l.attempts[key], now)

	// Opportunistic cleanup so the map cannot grow without bound on a
	// long-running process with many distinct source addresses.
	l.cleans++
	if l.cleans%100 == 0 {
		for k, times := range l.attempts {
			keep := times[:0]
			for _, t := range times {
				if t.After(cutoff) {
					keep = append(keep, t)
				}
			}
			if len(keep) == 0 {
				delete(l.attempts, k)
			} else {
				l.attempts[k] = keep
			}
		}
	}
	return true
}

// clientIP extracts the best-effort remote address for rate limiting.
// X-Forwarded-For is trusted only for the first hop; deployments behind a
// reverse proxy should configure it correctly or rely on the socket address.
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		first := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if first != "" {
			return first
		}
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		host = host[:idx]
	}
	return host
}
