package task

import (
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

func TestNewExpiresAtRequiresFutureInstant(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	future := NewExpiresAt(now.Add(time.Hour), now)
	accepted, matched := future.(ExpirationPolicyAccepted)
	if !matched {
		t.Fatalf("future instant rejected: %#v", future)
	}
	instant, instantMatched := accepted.Value.(ExpiresAt)
	if !instantMatched || !instant.Instant.Equal(now.Add(time.Hour)) {
		t.Fatalf("policy = %#v", accepted.Value)
	}

	past := NewExpiresAt(now.Add(-time.Second), now)
	rejected, rejectedMatched := past.(ExpirationPolicyRejected)
	if !rejectedMatched {
		t.Fatalf("past instant accepted: %#v", past)
	}
	if rejected.Reason.Code().String() != core.ErrorCodeInvalidArgument.String() {
		t.Fatalf("code = %s", rejected.Reason.Code().String())
	}

	exact := NewExpiresAt(now, now)
	if _, exactMatched := exact.(ExpirationPolicyRejected); !exactMatched {
		t.Fatalf("instant equal to now accepted: %#v", exact)
	}
}

func TestParseExpirationPolicy(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	none := ParseExpirationPolicy("", now)
	noneAccepted, noneMatched := none.(ExpirationPolicyAccepted)
	if !noneMatched {
		t.Fatalf("empty string rejected: %#v", none)
	}
	if _, isNone := noneAccepted.Value.(NoExpiration); !isNone {
		t.Fatalf("empty string parsed to %#v", noneAccepted.Value)
	}

	junk := ParseExpirationPolicy("tomorrow", now)
	if _, junkRejected := junk.(ExpirationPolicyRejected); !junkRejected {
		t.Fatalf("junk accepted: %#v", junk)
	}

	future := ParseExpirationPolicy("2026-07-02T12:00:00Z", now)
	futureAccepted, futureMatched := future.(ExpirationPolicyAccepted)
	if !futureMatched {
		t.Fatalf("future RFC3339 rejected: %#v", future)
	}
	if ExpirationInstantString(futureAccepted.Value) != "2026-07-02T12:00:00Z" {
		t.Fatalf("round trip = %q", ExpirationInstantString(futureAccepted.Value))
	}

	past := ParseExpirationPolicy("2026-06-30T12:00:00Z", now)
	if _, pastRejected := past.(ExpirationPolicyRejected); !pastRejected {
		t.Fatalf("past accepted: %#v", past)
	}
}

func TestExpirationInstantStringForNoExpiration(t *testing.T) {
	if ExpirationInstantString(NoExpiration{}) != "" {
		t.Fatalf("NoExpiration should render empty")
	}
}

func TestExpireStateOnlyFromOpen(t *testing.T) {
	accepted, matched := ExpireState(StateOpen).(StateTransitionAccepted)
	if !matched || accepted.Value != StateExpired {
		t.Fatalf("open should expire, got %#v", ExpireState(StateOpen))
	}
	for _, state := range []State{StateDraft, StateClosed, StateCancelled, StateExpired} {
		result := ExpireState(state)
		rejected, rejectedMatched := result.(StateTransitionRejected)
		if !rejectedMatched {
			t.Fatalf("state %s should not expire, got %#v", state.String(), result)
		}
		if rejected.Reason.Code().String() != core.ErrorCodeInvalidState.String() {
			t.Fatalf("code = %s", rejected.Reason.Code().String())
		}
	}
}
