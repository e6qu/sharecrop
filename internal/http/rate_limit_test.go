package httpserver

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterExhaustsAndRefills(t *testing.T) {
	current := time.Unix(0, 0).UTC()
	limiter := newRateLimiter(3, 1)
	limiter.now = func() time.Time { return current }

	for i := 0; i < 3; i++ {
		if !limiter.Allow("client") {
			t.Fatalf("request %d should be allowed within the burst capacity", i)
		}
	}
	if limiter.Allow("client") {
		t.Fatalf("request beyond the burst capacity should be denied")
	}

	// One second refills one token.
	current = current.Add(time.Second)
	if !limiter.Allow("client") {
		t.Fatalf("request should be allowed after a token refilled")
	}
	if limiter.Allow("client") {
		t.Fatalf("only one token should have refilled")
	}
}

func TestRateLimiterIsolatesKeys(t *testing.T) {
	current := time.Unix(0, 0).UTC()
	limiter := newRateLimiter(1, 1)
	limiter.now = func() time.Time { return current }

	if !limiter.Allow("a") {
		t.Fatalf("first request for key a should be allowed")
	}
	if limiter.Allow("a") {
		t.Fatalf("second request for key a should be denied")
	}
	if !limiter.Allow("b") {
		t.Fatalf("a different key must not share key a's bucket")
	}
}

func TestAuthenticationRateLimitIsolatesOperationsForOneClient(t *testing.T) {
	server := Server{ipRateLimiter: newRateLimiter(1, 1)}
	login := httptest.NewRequest("POST", "/api/auth/login", nil)
	login.RemoteAddr = "192.0.2.10:1234"
	if !server.allowByIP(httptest.NewRecorder(), login) {
		t.Fatal("first login should be allowed")
	}
	if server.allowByIP(httptest.NewRecorder(), login) {
		t.Fatal("second login should exhaust the login bucket")
	}

	register := httptest.NewRequest("POST", "/api/auth/register", nil)
	register.RemoteAddr = login.RemoteAddr
	if !server.allowByIP(httptest.NewRecorder(), register) {
		t.Fatal("an exhausted login bucket must not deny registration")
	}
}

func TestRateLimiterEvictsIdleBuckets(t *testing.T) {
	current := time.Unix(0, 0).UTC()
	limiter := newRateLimiter(2, 1)
	limiter.now = func() time.Time { return current }

	limiter.Allow("transient")
	if len(limiter.buckets) == 0 {
		t.Fatalf("an active bucket should be tracked")
	}

	// Advance well past the full-refill window and trigger the sweep.
	current = current.Add(10 * time.Second)
	limiter.Allow("other")
	if _, found := limiter.buckets["transient"]; found {
		t.Fatalf("a fully-refilled idle bucket should have been evicted")
	}
}
