package wasmdemo

import (
	"encoding/json"
	"testing"
	"time"
)

type fixedHandlerClock struct {
	value time.Time
}

func (clock fixedHandlerClock) Now() time.Time {
	return clock.value
}

func TestModerationTriageHandlerPersistsThroughBrowserStorage(t *testing.T) {
	storage := newTestBrowserStorage()
	handler := NewModerationTriageHandler(storage, fixedHandlerClock{value: time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)})
	result := handler.Handle(Request{
		Method: MethodPost,
		Path:   "/api/admin/moderation/reports/audit-1/triage",
		Body:   `{"state":"resolved","resolution_note":"handled","updated_by":"user-admin"}`,
	})
	handled, matched := result.(RequestHandled)
	if !matched {
		t.Fatalf("result = %T, want RequestHandled", result)
	}
	if handled.Value.Status != 200 {
		t.Fatalf("status = %d, want 200", handled.Value.Status)
	}
	var response StoredModerationTriage
	if err := json.Unmarshal([]byte(handled.Value.Body), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ReportID != "audit-1" || response.State != "resolved" || response.ResolutionNote != "handled" || response.UpdatedBy != "user-admin" {
		t.Fatalf("response = %#v", response)
	}

	loaded := LoadModerationTriage(storage, "audit-1")
	stored, storedMatched := loaded.(ModerationTriageStored)
	if !storedMatched {
		t.Fatalf("load result = %T, want ModerationTriageStored", loaded)
	}
	if stored.Value.UpdatedAt != "2026-06-30T12:00:00Z" {
		t.Fatalf("updated_at = %q", stored.Value.UpdatedAt)
	}
}

func TestModerationTriageHandlerRejectsUnsupportedRoute(t *testing.T) {
	handler := NewModerationTriageHandler(newTestBrowserStorage(), fixedHandlerClock{value: time.Now().UTC()})
	result := handler.Handle(Request{Method: MethodPost, Path: "/api/admin/privacy-retention/run", Body: `{}`})
	rejected, matched := result.(RequestHandleRejected)
	if !matched {
		t.Fatalf("result = %T, want RequestHandleRejected", result)
	}
	if rejected.Reason != "request route is not implemented by the WASM demo handler" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestModerationTriageHandlerRejectsMissingStorage(t *testing.T) {
	handler := NewModerationTriageHandler(nil, fixedHandlerClock{value: time.Now().UTC()})
	result := handler.Handle(Request{Method: MethodPost, Path: "/api/admin/moderation/reports/audit-1/triage", Body: `{}`})
	rejected, matched := result.(RequestHandleRejected)
	if !matched {
		t.Fatalf("result = %T, want RequestHandleRejected", result)
	}
	if rejected.Reason != "browser storage is required" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}
