package ratelimitbridge

import (
	"encoding/json"
	"errors"
	"testing"
)

// TestAllowFailsClosedOnTransportError proves the guest limiter DENIES when
// the bridge cannot reach the host, matching the host store's fail-closed
// behavior on a storage error: an outage must throttle, never un-throttle.
func TestAllowFailsClosedOnTransportError(t *testing.T) {
	limiter := NewGuestRateLimiter(func(string, []byte) ([]byte, error) {
		return nil, errors.New("bridge down")
	}, "ip")
	if limiter.Allow("198.51.100.7") {
		t.Fatalf("a failed bridge call must deny, not allow")
	}
}

// TestAllowFailsClosedOnGarbageReply covers the decode half of the failure
// surface: an unparseable host reply is a failure, not permission.
func TestAllowFailsClosedOnGarbageReply(t *testing.T) {
	limiter := NewGuestRateLimiter(func(string, []byte) ([]byte, error) {
		return []byte("not json"), nil
	}, "subject")
	if limiter.Allow("agent-1") {
		t.Fatalf("an undecodable reply must deny, not allow")
	}
}

// TestAllowPassesThroughHostDecision keeps the healthy path honest in both
// directions.
func TestAllowPassesThroughHostDecision(t *testing.T) {
	for _, decision := range []bool{true, false} {
		reply, err := json.Marshal(decision)
		if err != nil {
			t.Fatalf("marshal fixture: %v", err)
		}
		limiter := NewGuestRateLimiter(func(string, []byte) ([]byte, error) {
			return reply, nil
		}, "ip")
		if limiter.Allow("198.51.100.7") != decision {
			t.Fatalf("host decision %v was not passed through", decision)
		}
	}
}
