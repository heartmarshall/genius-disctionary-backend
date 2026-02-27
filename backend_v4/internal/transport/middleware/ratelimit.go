package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter implements per-IP token bucket rate limiting.
type RateLimiter struct {
	buckets sync.Map // map[string]*bucket
	stop    chan struct{}
}

type bucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a rate limiter with background cleanup.
// Call Stop() on shutdown.
func NewRateLimiter(cleanupInterval time.Duration) *RateLimiter {
	rl := &RateLimiter{stop: make(chan struct{})}
	go rl.cleanup(cleanupInterval)
	return rl
}

// Stop terminates the background cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stop)
}

// Limit returns middleware that rate-limits requests to maxPerMinute per IP.
func (rl *RateLimiter) Limit(maxPerMinute int) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			b := rl.getBucket(ip, maxPerMinute)
			if !b.allow() {
				retryAfter := 60.0 / float64(maxPerMinute)
				w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter)+1))
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) getBucket(key string, maxPerMinute int) *bucket {
	maxTokens := float64(maxPerMinute)
	refillRate := maxTokens / 60.0

	val, _ := rl.buckets.LoadOrStore(key, &bucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	})

	return val.(*bucket)
}

func (b *bucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
	b.lastRefill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (rl *RateLimiter) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stop:
			return
		case <-ticker.C:
			now := time.Now()
			rl.buckets.Range(func(key, value any) bool {
				b := value.(*bucket)
				b.mu.Lock()
				idle := now.Sub(b.lastRefill)
				b.mu.Unlock()
				if idle > 10*time.Minute {
					rl.buckets.Delete(key)
				}
				return true
			})
		}
	}
}
