package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"

	"url-shortener/internal/transport/http/response"
)

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// Limiter is a per-key token bucket. rate is tokens/sec refilled, burst
// is the bucket capacity (and therefore the max instantaneous burst
// allowed).
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64
	burst   float64
	ttl     time.Duration
}

// New creates a Limiter allowing `rate` requests/sec per key, with bursts
// up to `burst`. Entries idle longer than ttl are evicted by a background
// sweeper to bound memory use under a large number of distinct IPs.
func New(rate float64, burst float64, ttl time.Duration) *Limiter {
	l := &Limiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		burst:   burst,
		ttl:     ttl,
	}
	go l.sweep()
	return l
}

func (l *Limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, lastSeen: now}
		l.buckets[key] = b
	}

	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens = min(l.burst, b.tokens+elapsed*l.rate)
	b.lastSeen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *Limiter) sweep() {
	ticker := time.NewTicker(l.ttl)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-l.ttl)
		l.mu.Lock()
		for key, b := range l.buckets {
			if b.lastSeen.Before(cutoff) {
				delete(l.buckets, key)
			}
		}
		l.mu.Unlock()
	}
}

// Middleware rate-limits by client IP (parsed from RemoteAddr — put this
// behind a trusted proxy that sets RemoteAddr correctly, or extend the
// key function to trust X-Forwarded-For only from known proxy hops).
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientIP(r)

		if !l.allow(key) {
			response.Respond(w, r, http.StatusTooManyRequests, response.Error("rate limit exceeded"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
