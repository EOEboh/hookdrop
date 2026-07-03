package middleware

import (
	"log"
	"sync"
	"time"
)

// EmailRateLimiter limits requests per email address within a sliding hour.
// Used to stop the magic-link endpoint from being used to spam a single
// inbox, or to exhaust the Resend send quota via repeated requests.
type EmailRateLimiter struct {
	mu         sync.Mutex
	attempts   map[string]*emailWindow
	maxPerHour int
}

type emailWindow struct {
	count       int
	windowStart time.Time
}

func NewEmailRateLimiter(maxPerHour int) *EmailRateLimiter {
	rl := &EmailRateLimiter{
		attempts:   make(map[string]*emailWindow),
		maxPerHour: maxPerHour,
	}
	go rl.cleanup()
	return rl
}

// Allow returns false once an email has exceeded maxPerHour requests
// within a rolling 1-hour window starting from its first request.
func (rl *EmailRateLimiter) Allow(email string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	w, ok := rl.attempts[email]

	if !ok || now.Sub(w.windowStart) > time.Hour {
		rl.attempts[email] = &emailWindow{count: 1, windowStart: now}
		return true
	}

	if w.count >= rl.maxPerHour {
		return false
	}

	w.count++
	return true
}

// cleanup prevents unbounded memory growth from one-off email addresses
func (rl *EmailRateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-1 * time.Hour)
		removed := 0
		for email, w := range rl.attempts {
			if w.windowStart.Before(cutoff) {
				delete(rl.attempts, email)
				removed++
			}
		}
		rl.mu.Unlock()
		if removed > 0 {
			log.Printf("email rate limiter: cleaned up %d stale entries", removed)
		}
	}
}
