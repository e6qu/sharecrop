package agentwire

import (
	"testing"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
)

func TestLabelStateScopeSetRoundTrip(t *testing.T) {
	label, matched := agent.NewLabel("shared-label").(agent.LabelAccepted)
	if !matched {
		t.Fatalf("label rejected")
	}
	restoredLabel, err := DecodeLabel(EncodeLabel(label.Value))
	if err != nil || restoredLabel.String() != "shared-label" {
		t.Errorf("label did not round-trip: %v %q", err, restoredLabel)
	}

	restoredState, err := DecodeState(EncodeState(agent.StateRevoked))
	if err != nil || restoredState.String() != agent.StateRevoked.String() {
		t.Errorf("state did not round-trip: %v", err)
	}

	scope, matched := agent.ParseScope("tasks_read").(agent.ScopeAccepted)
	if !matched {
		t.Fatalf("scope rejected")
	}
	restoredScopes, err := DecodeScopeSet(EncodeScopeSet(agent.NewScopeSet([]agent.Scope{scope.Value})))
	if err != nil {
		t.Fatalf("decode scopes: %v", err)
	}
	if values := restoredScopes.Values(); len(values) != 1 || values[0].String() != "tasks_read" {
		t.Errorf("scope set did not round-trip: %+v", values)
	}
	if _, err := DecodeScopeSet([]string{"not-a-scope"}); err == nil {
		t.Errorf("DecodeScopeSet accepted a bad scope")
	}
}

func TestCreateStoreResultRoundTrip(t *testing.T) {
	accepted, err := DecodeCreateStoreResult(EncodeCreateStoreResult(agent.CreateStoreAccepted{}))
	if err != nil {
		t.Fatalf("decode accepted: %v", err)
	}
	if _, matched := accepted.(agent.CreateStoreAccepted); !matched {
		t.Errorf("accepted result = %T", accepted)
	}
	rejected, err := DecodeCreateStoreResult(EncodeCreateStoreResult(agent.CreateStoreRejected{
		Reason: core.NewDomainError(core.ErrorCodeConflict, "dup"),
	}))
	if err != nil {
		t.Fatalf("decode rejected: %v", err)
	}
	if typed, matched := rejected.(agent.CreateStoreRejected); !matched || typed.Reason.Code() != core.ErrorCodeConflict {
		t.Errorf("create rejection not preserved: %T", rejected)
	}
}
