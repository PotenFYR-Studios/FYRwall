package auth

import (
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// clientKey identifies a rate-limit bucket: remote IP (and optionally the
// username for targeted lockout on credential stuffing).
type bucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// LoginRateLimiter bounds login attempts per source IP (spec sections 13,
// 35, 77). Buckets expire after inactivity to bound memory.
type LoginRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	perMin  rate.Limit
	burst   int
	maxAge  time.Duration
}

// NewLoginRateLimiter allows perMinute attempts per minute with burst 2.
func NewLoginRateLimiter(perMinute int) *LoginRateLimiter {
	if perMinute < 1 {
		perMinute = 5
	}
	return &LoginRateLimiter{
		buckets: map[string]*bucket{},
		perMin:  rate.Limit(float64(perMinute) / 60.0),
		burst:   2,
		maxAge:  30 * time.Minute,
	}
}

// Allow reports whether a login attempt from ip is permitted.
func (l *LoginRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.gc()
	b, ok := l.buckets[ip]
	if !ok {
		b = &bucket{limiter: rate.NewLimiter(l.perMin, l.burst), lastSeen: time.Now()}
		l.buckets[ip] = b
	}
	b.lastSeen = time.Now()
	return b.limiter.Allow()
}

// gc drops stale buckets; caller holds the lock.
func (l *LoginRateLimiter) gc() {
	now := time.Now()
	for k, b := range l.buckets {
		if now.Sub(b.lastSeen) > l.maxAge {
			delete(l.buckets, k)
		}
	}
}

// NormalizeIP extracts the client IP for rate limiting, honoring
// X-Forwarded-For only from trusted proxies (handled by chi middleware
// upstream; this just takes the last-hop RemoteAddr host).
func NormalizeIP(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}
