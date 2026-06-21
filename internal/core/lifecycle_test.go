package core

import "testing"

func TestParseLifecycleState(t *testing.T) {
	result := ParseLifecycleState("active")

	parsed, matched := result.(LifecycleStateParsed)
	if !matched {
		t.Fatalf("result = %T, want LifecycleStateParsed", result)
	}

	if parsed.Value.String() != "active" {
		t.Fatalf("state = %q, want active", parsed.Value.String())
	}
}

func TestParseLifecycleStateRejectsUnknownValue(t *testing.T) {
	result := ParseLifecycleState("deleted")

	_, matched := result.(LifecycleStateRejected)
	if !matched {
		t.Fatalf("result = %T, want LifecycleStateRejected", result)
	}
}
