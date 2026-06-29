package httpserver

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type RateLimiter interface {
	Allow(key string) bool
	ActiveBuckets() int
	StorageKind() string
}

// rateLimiter is an in-memory token-bucket limiter keyed by an arbitrary string
// (a client IP for unauthenticated endpoints, an agent subject for MCP). It is a
// defense-in-depth/availability control, not a distributed quota; like the MCP
// session store it lives per process. Idle buckets are evicted so keys cannot
// accumulate without bound.
type rateLimiter struct {
	mu           sync.Mutex
	buckets      map[string]rateBucket
	capacity     float64
	refillPerSec float64
	fullAfter    float64
	lastSweep    time.Time
	now          func() time.Time
}

type rateBucket struct {
	tokens  float64
	updated time.Time
}

func newRateLimiter(capacity int, refillPerSec float64) *rateLimiter {
	return &rateLimiter{
		buckets:      make(map[string]rateBucket),
		capacity:     float64(capacity),
		refillPerSec: refillPerSec,
		fullAfter:    float64(capacity) / refillPerSec,
		now:          time.Now,
	}
}

// Allow consumes one token for key and reports whether the request is permitted.
func (limiter *rateLimiter) Allow(key string) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := limiter.now()
	if now.Sub(limiter.lastSweep).Seconds() >= limiter.fullAfter {
		limiter.evictFullLocked(now)
		limiter.lastSweep = now
	}

	tokens := limiter.capacity
	if bucket, found := limiter.buckets[key]; found {
		tokens = bucket.tokens + now.Sub(bucket.updated).Seconds()*limiter.refillPerSec
		if tokens > limiter.capacity {
			tokens = limiter.capacity
		}
	}
	if tokens < 1 {
		limiter.buckets[key] = rateBucket{tokens: tokens, updated: now}
		return false
	}
	limiter.buckets[key] = rateBucket{tokens: tokens - 1, updated: now}
	return true
}

// evictFullLocked drops buckets that have had time to fully refill; a fully
// refilled bucket is indistinguishable from an absent one, so removing it bounds
// the map to recently-active keys. Callers must hold the mutex.
func (limiter *rateLimiter) evictFullLocked(now time.Time) {
	for key, bucket := range limiter.buckets {
		if now.Sub(bucket.updated).Seconds() >= limiter.fullAfter {
			delete(limiter.buckets, key)
		}
	}
}

func (limiter *rateLimiter) ActiveBuckets() int {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	limiter.evictFullLocked(limiter.now())
	return len(limiter.buckets)
}

func (limiter *rateLimiter) StorageKind() string {
	return "process_memory"
}

// clientIP returns the direct peer address (host without port). It intentionally
// does not trust X-Forwarded-For, which a client can forge to evade the limit.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
