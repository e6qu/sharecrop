package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
)

// eventFeedTestHandler builds the real mux over an in-memory event store
// seeded with two events visible to the authenticated test user and one
// visible only to a stranger.
func eventFeedTestHandler(t *testing.T) (http.Handler, []event.StoredEvent) {
	t.Helper()
	store := event.NewMemoryStore()
	stranger := core.NewUserID().(core.UserIDCreated).Value

	visible := make([]event.StoredEvent, 0, 2)
	for _, kind := range []event.Kind{event.KindTaskOpened, event.KindReservationRequested} {
		appended, matched := store.Append(context.Background(), testFeedEvent(t, kind), event.NewRecipients(stableTestUserID)).(event.AppendStoreAccepted)
		if !matched {
			t.Fatalf("seed visible event rejected")
		}
		visible = append(visible, appended.Value)
	}
	if _, matched := store.Append(context.Background(), testFeedEvent(t, event.KindTaskFunded), event.NewRecipients(stranger)).(event.AppendStoreAccepted); !matched {
		t.Fatalf("seed stranger event rejected")
	}

	runtime := DefaultRuntimeState(map[string]bool{})
	runtime.EventStore = store
	return NewWithRuntimeState(testStaticFiles(), testAuthService(), testVerifier{}, testOrganizationService{}, testTaskService{}, testSubmissionService{}, testLedgerService{}, testAgentService{}, testOrgCredentialService{}, testAssetService{}, runtime), visible
}

func testFeedEvent(t *testing.T, kind event.Kind) event.Event {
	t.Helper()
	id, matched := core.NewDomainEventID().(core.DomainEventIDCreated)
	if !matched {
		t.Fatalf("domain event id rejected")
	}
	subject := event.NoSubjectRefs()
	subject.Task = event.TaskSubject{ID: core.NewTaskID().(core.TaskIDCreated).Value}
	return event.Event{
		ID:       id.Value,
		Kind:     kind,
		Actor:    event.ActorUser{ID: stableTestUserID},
		Subject:  subject,
		Metadata: event.EmptyMetadata(),
	}
}

type decodedEventListResponse struct {
	Events []struct {
		Kind        string `json:"kind"`
		ActorKind   string `json:"actor_kind"`
		ActorUserID string `json:"actor_user_id"`
		Cursor      string `json:"cursor"`
		TaskID      string `json:"task_id"`
	} `json:"events"`
	NextCursor string `json:"next_cursor"`
}

func fetchEventFeed(t *testing.T, handler http.Handler, target string) decodedEventListResponse {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusOK, response.Body.String())
	}
	var body decodedEventListResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body
}

func TestEventFeedListsOnlyTheCallersEvents(t *testing.T) {
	handler, visible := eventFeedTestHandler(t)

	body := fetchEventFeed(t, handler, "/api/events")
	if len(body.Events) != 2 {
		t.Fatalf("feed returned %d events, want 2: %+v", len(body.Events), body)
	}
	if body.Events[0].Kind != "task_opened" || body.Events[1].Kind != "reservation_requested" {
		t.Fatalf("feed kinds = %+v", body.Events)
	}
	if body.Events[0].ActorKind != "user" || body.Events[0].ActorUserID != stableTestUserID.String() {
		t.Fatalf("feed actor = %+v", body.Events[0])
	}
	if body.Events[0].TaskID == "" {
		t.Fatalf("feed event lost its task reference: %+v", body.Events[0])
	}
	if body.NextCursor != visible[1].Cursor.String() {
		t.Fatalf("next_cursor = %q, want %q", body.NextCursor, visible[1].Cursor.String())
	}
}

func TestEventFeedAfterCursorExcludesOlderEvents(t *testing.T) {
	handler, visible := eventFeedTestHandler(t)

	body := fetchEventFeed(t, handler, "/api/events?after="+visible[0].Cursor.String())
	if len(body.Events) != 1 {
		t.Fatalf("feed after cursor returned %d events, want 1: %+v", len(body.Events), body)
	}
	if body.Events[0].Cursor != visible[1].Cursor.String() {
		t.Fatalf("feed after cursor returned %+v", body.Events[0])
	}
}

func TestEventFeedEmptyPageHasEmptyNextCursor(t *testing.T) {
	handler, visible := eventFeedTestHandler(t)

	body := fetchEventFeed(t, handler, "/api/events?after="+visible[1].Cursor.String())
	if len(body.Events) != 0 || body.NextCursor != "" {
		t.Fatalf("empty page = %+v, want no events and empty next_cursor", body)
	}
}

func TestEventFeedRejectsMalformedQuery(t *testing.T) {
	handler, _ := eventFeedTestHandler(t)

	for _, target := range []string{"/api/events?after=nonsense", "/api/events?limit=zero", "/api/events?limit=0"} {
		request := httptest.NewRequest(http.MethodGet, target, nil)
		request.Header.Set("Authorization", "Bearer test-access-token")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want %d", target, response.Code, http.StatusBadRequest)
		}
	}
}

func TestEventFeedRequiresAuth(t *testing.T) {
	handler, _ := eventFeedTestHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

// nonFlushingRecorder hides httptest.ResponseRecorder's Flush method so the
// SSE handler takes the bounded replay-and-return branch, exactly as it does
// behind the WASI request bridge.
type nonFlushingRecorder struct {
	recorder *httptest.ResponseRecorder
}

func (w nonFlushingRecorder) Header() http.Header {
	return w.recorder.Header()
}

func (w nonFlushingRecorder) Write(data []byte) (int, error) {
	return w.recorder.Write(data)
}

func (w nonFlushingRecorder) WriteHeader(status int) {
	w.recorder.WriteHeader(status)
}

func TestEventStreamReplaysEventsWithCursorIDs(t *testing.T) {
	handler, visible := eventFeedTestHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/events/stream", nil)
	request.Header.Set("Authorization", "Bearer test-access-token")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(nonFlushingRecorder{recorder: recorder}, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (%s)", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Fatalf("content type = %q, want text/event-stream", contentType)
	}
	body := recorder.Body.String()
	for _, stored := range visible {
		if !strings.Contains(body, "id: "+stored.Cursor.String()+"\n") {
			t.Fatalf("stream body is missing id line for cursor %s: %q", stored.Cursor.String(), body)
		}
	}
	if !strings.Contains(body, `"kind":"reservation_requested"`) {
		t.Fatalf("stream body is missing the event JSON: %q", body)
	}
}

func TestEventStreamHonorsLastEventIDOverAfter(t *testing.T) {
	handler, visible := eventFeedTestHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/events/stream?after=0", nil)
	request.Header.Set("Authorization", "Bearer test-access-token")
	request.Header.Set("Last-Event-ID", visible[0].Cursor.String())
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(nonFlushingRecorder{recorder: recorder}, request)

	body := recorder.Body.String()
	if strings.Contains(body, "id: "+visible[0].Cursor.String()+"\n") {
		t.Fatalf("stream replayed the event before Last-Event-ID: %q", body)
	}
	if !strings.Contains(body, "id: "+visible[1].Cursor.String()+"\n") {
		t.Fatalf("stream is missing the event after Last-Event-ID: %q", body)
	}
}

func TestEventStreamStopsWhenTheClientDisconnects(t *testing.T) {
	handler, _ := eventFeedTestHandler(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodGet, "/api/events/stream", nil).WithContext(ctx)
	request.Header.Set("Authorization", "Bearer test-access-token")
	recorder := httptest.NewRecorder()
	// The recorder implements http.Flusher, so the handler enters the live
	// polling loop; the pre-cancelled context must end it immediately.
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}
