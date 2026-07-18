package main

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// ipLimiter rate-limits per client IP with a simple in-memory map. Entries are
// evicted lazily once stale so the map cannot grow unbounded.
type ipLimiter struct {
	mu       sync.Mutex
	clients  map[string]*client
	limit    rate.Limit
	burst    int
	maxIdle  time.Duration
	lastScan time.Time
}

type client struct {
	limiter *rate.Limiter
	seen    time.Time
}

// newIPLimiter allows `perHour` requests per IP with the given burst.
func newIPLimiter(perHour, burst int) *ipLimiter {
	return &ipLimiter{
		clients: make(map[string]*client),
		limit:   rate.Every(time.Hour / time.Duration(perHour)),
		burst:   burst,
		maxIdle: time.Hour,
	}
}

// newGlobalLimiter caps total outbound emails across all clients — a backstop
// against distributed abuse that slips past the per-IP limit.
func newGlobalLimiter(perHour, burst int) *rate.Limiter {
	return rate.NewLimiter(rate.Every(time.Hour/time.Duration(perHour)), burst)
}

func (l *ipLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if now.Sub(l.lastScan) > l.maxIdle {
		for k, c := range l.clients {
			if now.Sub(c.seen) > l.maxIdle {
				delete(l.clients, k)
			}
		}
		l.lastScan = now
	}

	c, ok := l.clients[ip]
	if !ok {
		c = &client{limiter: rate.NewLimiter(l.limit, l.burst)}
		l.clients[ip] = c
	}
	c.seen = now
	return c.limiter.Allow()
}
