package middleware

import (
	"log"
	"net/http"
	"sync"
	"time"
)

type bucket struct {
	tokens    float64
	lastRefil time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	capacity float64 // max tokens per IP
	refill   float64 // tokens added per second
}

func NewRateLimiter(capacity, refillPerSecond float64) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*bucket),
		capacity: capacity,
		refill:   refillPerSecond,
	}
	// Background goroutine cleans up stale IPs every 5 minutes
	// Prevents unbounded memory growth from one-off IPs
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[ip]
	if !ok {
		// First request from this IP: full bucket
		rl.buckets[ip] = &bucket{
			tokens:    rl.capacity - 1,
			lastRefil: time.Now(),
		}
		return true
	}

	// Refill based on elapsed time since last request
	now := time.Now()
	elapsed := now.Sub(b.lastRefil).Seconds()
	b.tokens = min(rl.capacity, b.tokens+elapsed*rl.refill)
	b.lastRefil = now

	if b.tokens < 1 {
		return false // bucket empty: reject
	}

	b.tokens--
	return true
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-10 * time.Minute)
		removed := 0
		for ip, b := range rl.buckets {
			if b.lastRefil.Before(cutoff) {
				delete(rl.buckets, ip)
				removed++
			}
		}
		if removed > 0 {
			log.Printf("rate limiter: cleaned up %d stale IP buckets", removed)
		}
		rl.mu.Unlock()
	}
}

// InboxRateLimit wraps a handler with per-IP rate limiting
// Extracts the real IP even behind proxies
func InboxRateLimit(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractClientIP(r)

			if !rl.allow(ip) {
				log.Printf("rate limit exceeded: ip=%s path=%s", ip, r.URL.Path)
				w.Header().Set("Retry-After", "10")
				http.Error(w, `{"error":"rate limit exceeded","retry_after":10}`,
					http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For first: present when behind Nginx/proxy
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {

		for _, part := range splitAndTrim(fwd, ',') {
			if part != "" {
				return part
			}
		}
	}
	// X-Real-IP set by Nginx
	if real := r.Header.Get("X-Real-IP"); real != "" {
		return real
	}
	// Fall back to remote addr: strip the port
	ip := r.RemoteAddr
	if idx := lastIndex(ip, ':'); idx != -1 {
		return ip[:idx]
	}
	return ip
}

func splitAndTrim(s string, sep rune) []string {
	var parts []string
	start := 0
	for i, c := range s {
		if c == sep {
			parts = append(parts, trim(s[start:i]))
			start = i + 1
		}
	}
	parts = append(parts, trim(s[start:]))
	return parts
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func lastIndex(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// AuthIPRateLimit wraps an http.HandlerFunc (rather than http.Handler)
// since auth routes are registered with mux.HandleFunc, not mux.Handle.
func AuthIPRateLimit(rl *RateLimiter) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := extractClientIP(r)

			if !rl.allow(ip) {
				log.Printf("auth rate limit exceeded: ip=%s path=%s", ip, r.URL.Path)
				w.Header().Set("Retry-After", "60")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"too many requests, please slow down"}`))
				return
			}

			next(w, r)
		}
	}
}
