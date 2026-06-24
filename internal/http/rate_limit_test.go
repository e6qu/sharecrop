package httpserver

import (
	"testing"
	"time"
)

func TestRateLimiterExhaustsAndRefills(t *testing.T) {
	current := time.Unix(0, 0).UTC()
	limiter := newRateLimiter(3, 1)
	limiter.now = func() time.Time { return current }

	for i := 0; i < 3; i++ {
		if !limiter.allow("client") {
			t.Fatalf("request %d should be allowed within the burst capacity", i)
		}
	}
	if limiter.allow("client") {
		t.Fatalf("request beyond the burst capacity should be denied")
	}

	// One second refills one token.
	current = current.Add(time.Second)
	if !limiter.allow("client") {
		t.Fatalf("request should be allowed after a token refilled")
	}
	if limiter.allow("client") {
		t.Fatalf("only one token should have refilled")
	}
}

func TestRateLimiterIsolatesKeys(t *testing.T) {
	current := time.Unix(0, 0).UTC()
	limiter := newRateLimiter(1, 1)
	limiter.now = func() time.Time { return current }

	if !limiter.allow("a") {
		t.Fatalf("first request for key a should be allowed")
	}
	if limiter.allow("a") {
		t.Fatalf("second request for key a should be denied")
	}
	if !limiter.allow("b") {
		t.Fatalf("a different key must not share key a's bucket")
	}
}

func TestRateLimiterEvictsIdleBuckets(t *testing.T) {
	current := time.Unix(0, 0).UTC()
	limiter := newRateLimiter(2, 1)
	limiter.now = func() time.Time { return current }

	limiter.allow("transient")
	if len(limiter.buckets) == 0 {
		t.Fatalf("an active bucket should be tracked")
	}

	// Advance well past the full-refill window and trigger the sweep.
	current = current.Add(10 * time.Second)
	limiter.allow("other")
	if _, found := limiter.buckets["transient"]; found {
		t.Fatalf("a fully-refilled idle bucket should have been evicted")
	}
}
