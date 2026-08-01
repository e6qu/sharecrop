package httpserver

import (
	"net/http"
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

func TestParseRegistrationRateCapacityForRuntime(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{raw: "", want: RegistrationRateCapacity},
		{raw: "  ", want: RegistrationRateCapacity},
		{raw: "100000", want: 100000},
		{raw: " 7 ", want: 7},
		{raw: "0", want: RegistrationRateCapacity},
		{raw: "-3", want: RegistrationRateCapacity},
		{raw: "plenty", want: RegistrationRateCapacity},
	}
	for _, testCase := range cases {
		if got := ParseRegistrationRateCapacityForRuntime(testCase.raw); got != testCase.want {
			t.Fatalf("ParseRegistrationRateCapacityForRuntime(%q) = %d, want %d", testCase.raw, got, testCase.want)
		}
	}
}

func TestParseRegistrationRateRefillForRuntime(t *testing.T) {
	cases := []struct {
		raw  string
		want float64
	}{
		{raw: "", want: RegistrationRateRefillPerSec},
		{raw: "  ", want: RegistrationRateRefillPerSec},
		{raw: "1000", want: 1000},
		{raw: " 0.5 ", want: 0.5},
		{raw: "0", want: RegistrationRateRefillPerSec},
		{raw: "-1", want: RegistrationRateRefillPerSec},
		{raw: "fast", want: RegistrationRateRefillPerSec},
	}
	for _, testCase := range cases {
		if got := ParseRegistrationRateRefillForRuntime(testCase.raw); got != testCase.want {
			t.Fatalf("ParseRegistrationRateRefillForRuntime(%q) = %g, want %g", testCase.raw, got, testCase.want)
		}
	}
}

func TestRegistrationRateLimitIsDedicatedPerIP(t *testing.T) {
	server := Server{registrationLimiter: newRateLimiter(2, RegistrationRateRefillPerSec)}
	request := httptest.NewRequest("POST", "/api/auth/register", nil)
	request.RemoteAddr = "192.0.2.20:1234"

	for i := 0; i < 2; i++ {
		if !server.allowRegistration(httptest.NewRecorder(), request) {
			t.Fatalf("registration attempt %d should be allowed within the burst capacity", i)
		}
	}
	if server.allowRegistration(httptest.NewRecorder(), request) {
		t.Fatal("registration attempt beyond the burst capacity should be denied")
	}

	other := httptest.NewRequest("POST", "/api/auth/register", nil)
	other.RemoteAddr = "192.0.2.21:1234"
	if !server.allowRegistration(httptest.NewRecorder(), other) {
		t.Fatal("another client's registration must not share the exhausted bucket")
	}
}

// TestRegisterHandlerAppliesRegistrationBudget proves the register handler
// itself returns 429 from the dedicated registration limiter even while the
// generic unauthenticated IP bucket still has budget.
func TestRegisterHandlerAppliesRegistrationBudget(t *testing.T) {
	server := Server{
		ipRateLimiter:       newRateLimiter(100, 100),
		registrationLimiter: newRateLimiter(1, RegistrationRateRefillPerSec),
	}
	request := httptest.NewRequest("POST", "/api/auth/register", nil)
	request.RemoteAddr = "192.0.2.22:1234"

	// The first attempt consumes the only token; it fails later at request
	// decoding (400), which is fine - the limiter ran first.
	first := httptest.NewRecorder()
	server.register(first, request)
	if first.Code == http.StatusTooManyRequests {
		t.Fatalf("first registration attempt must not be rate limited, got %d", first.Code)
	}

	second := httptest.NewRecorder()
	server.register(second, request)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second registration attempt = %d, want 429 from the dedicated registration budget", second.Code)
	}
}
