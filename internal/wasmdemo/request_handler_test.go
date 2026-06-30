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

type fixedHandlerActor struct {
	userID string
}

func (actor fixedHandlerActor) UserID() string {
	return actor.userID
}

type fixedPrivacyRequestIDs struct {
	value string
}

func (ids fixedPrivacyRequestIDs) NextPrivacyRequestID() string {
	return ids.value
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

func TestPrivacyRequestHandlerCreatesAndListsThroughBrowserStorage(t *testing.T) {
	storage := newTestBrowserStorage()
	handler := NewPrivacyRequestHandler(
		storage,
		fixedHandlerClock{value: time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)},
		fixedHandlerActor{userID: "user-admin"},
		fixedPrivacyRequestIDs{value: "privacy-1"},
	)

	createResult := handler.Handle(Request{
		Method: MethodPost,
		Path:   "/api/privacy-requests",
		Body:   `{"kind":"data_export"}`,
	})
	created, createdMatched := createResult.(RequestHandled)
	if !createdMatched {
		t.Fatalf("create result = %T, want RequestHandled", createResult)
	}
	if created.Value.Status != 201 {
		t.Fatalf("create status = %d, want 201", created.Value.Status)
	}
	var createdBody StoredPrivacyRequest
	if err := json.Unmarshal([]byte(created.Value.Body), &createdBody); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createdBody.ID != "privacy-1" || createdBody.Kind != "data_export" || createdBody.Status != "queued" || createdBody.RequestedBy != "user-admin" {
		t.Fatalf("created body = %#v", createdBody)
	}

	listResult := handler.Handle(Request{Method: MethodGet, Path: "/api/admin/privacy-requests", Body: ""})
	listed, listedMatched := listResult.(RequestHandled)
	if !listedMatched {
		t.Fatalf("list result = %T, want RequestHandled", listResult)
	}
	if listed.Value.Status != 200 {
		t.Fatalf("list status = %d, want 200", listed.Value.Status)
	}
	var listBody privacyRequestsBody
	if err := json.Unmarshal([]byte(listed.Value.Body), &listBody); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listBody.Requests) != 1 || listBody.Requests[0].ID != "privacy-1" {
		t.Fatalf("list body = %#v", listBody)
	}
}

func TestPrivacyRequestHandlerRejectsMissingIDSource(t *testing.T) {
	handler := NewPrivacyRequestHandler(
		newTestBrowserStorage(),
		fixedHandlerClock{value: time.Now().UTC()},
		fixedHandlerActor{userID: "user-admin"},
		nil,
	)
	result := handler.Handle(Request{Method: MethodPost, Path: "/api/privacy-requests", Body: `{"kind":"data_export"}`})
	rejected, matched := result.(RequestHandleRejected)
	if !matched {
		t.Fatalf("result = %T, want RequestHandleRejected", result)
	}
	if rejected.Reason != "privacy request id source is required" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestPrivacyRequestHandlerRejectsInvalidKind(t *testing.T) {
	handler := NewPrivacyRequestHandler(
		newTestBrowserStorage(),
		fixedHandlerClock{value: time.Now().UTC()},
		fixedHandlerActor{userID: "user-admin"},
		fixedPrivacyRequestIDs{value: "privacy-1"},
	)
	result := handler.Handle(Request{Method: MethodPost, Path: "/api/privacy-requests", Body: `{"kind":"unknown"}`})
	rejected, matched := result.(RequestHandleRejected)
	if !matched {
		t.Fatalf("result = %T, want RequestHandleRejected", result)
	}
	if rejected.Reason != "privacy request kind is invalid" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}
