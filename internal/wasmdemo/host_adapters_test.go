package wasmdemo

import (
	"testing"
	"time"
)

type fixedHostRuntime struct {
	storage        BrowserStorage
	clock          HandlerClock
	actor          HandlerActor
	interactionIDs InteractionIDSource
}

func (runtime fixedHostRuntime) Storage() BrowserStorage {
	return runtime.storage
}

func (runtime fixedHostRuntime) Clock() HandlerClock {
	return runtime.clock
}

func (runtime fixedHostRuntime) Actor() HandlerActor {
	return runtime.actor
}

func (runtime fixedHostRuntime) InteractionIDs() InteractionIDSource {
	return runtime.interactionIDs
}

func TestNewHostRequestAcceptsSupportedRequest(t *testing.T) {
	result := NewHostRequest("POST", "/api/tasks/task-1/submissions", `{"response_json":"{}"}`)
	accepted, matched := result.(HostRequestAccepted)
	if !matched {
		t.Fatalf("result = %T, want HostRequestAccepted", result)
	}
	if accepted.Value.Method != "POST" || accepted.Value.Path != "/api/tasks/task-1/submissions" {
		t.Fatalf("request = %#v", accepted.Value)
	}
}

func TestNewHostRequestRejectsUnsupportedMethod(t *testing.T) {
	result := NewHostRequest("DELETE", "/api/tasks/task-1/submissions", "")
	rejected, matched := result.(HostRequestRejected)
	if !matched {
		t.Fatalf("result = %T, want HostRequestRejected", result)
	}
	if rejected.Reason != "request method is unsupported" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestValidateHostRuntimeRequiresAdapters(t *testing.T) {
	result := ValidateHostRuntime(fixedHostRuntime{})
	rejected, matched := result.(HostRuntimeRejected)
	if !matched {
		t.Fatalf("result = %T, want HostRuntimeRejected", result)
	}
	if rejected.Reason != "host storage adapter is required" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestValidateHostRuntimeAcceptsExplicitAdapters(t *testing.T) {
	result := ValidateHostRuntime(fixedHostRuntime{
		storage:        newTestBrowserStorage(),
		clock:          fixedHandlerClock{value: time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)},
		actor:          fixedHandlerActor{userID: "user-1"},
		interactionIDs: fixedInteractionIDs{submissionID: "submission-1", commentID: "comment-1", reservationID: "reservation-1", ledgerID: "ledger-1"},
	})
	accepted, matched := result.(HostRuntimeAccepted)
	if !matched {
		t.Fatalf("result = %T, want HostRuntimeAccepted", result)
	}
	if accepted.Storage == nil || accepted.Clock == nil || accepted.Actor == nil || accepted.InteractionIDs == nil {
		t.Fatalf("accepted runtime has missing adapters: %#v", accepted)
	}
}
