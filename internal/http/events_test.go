package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/orgcred"
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

// ---- long-poll (?wait=) ----

func TestParseEventFeedWaitBounds(t *testing.T) {
	cases := []struct {
		raw  string
		want time.Duration
	}{
		{raw: "", want: 0},
		{raw: "0", want: 0},
		{raw: "3", want: 3 * time.Second},
		{raw: "25", want: 25 * time.Second},
		{raw: "26", want: 25 * time.Second},
		{raw: "999", want: 25 * time.Second},
	}
	for _, testCase := range cases {
		request := httptest.NewRequest(http.MethodGet, "/api/events?wait="+testCase.raw, nil)
		if testCase.raw == "" {
			request = httptest.NewRequest(http.MethodGet, "/api/events", nil)
		}
		accepted, matched := parseEventFeedWait(request).(eventFeedWaitAccepted)
		if !matched {
			t.Fatalf("wait=%q rejected, want accepted", testCase.raw)
		}
		if accepted.value != testCase.want {
			t.Fatalf("wait=%q parsed to %v, want %v", testCase.raw, accepted.value, testCase.want)
		}
	}
	for _, raw := range []string{"-1", "two", "1.5"} {
		if _, matched := parseEventFeedWait(httptest.NewRequest(http.MethodGet, "/api/events?wait="+raw, nil)).(eventFeedWaitRejected); !matched {
			t.Fatalf("wait=%q accepted, want rejected", raw)
		}
	}
}

func TestEventFeedRejectsMalformedWait(t *testing.T) {
	handler, _ := eventFeedTestHandler(t)

	for _, target := range []string{"/api/events?wait=-1", "/api/events?wait=soon"} {
		request := httptest.NewRequest(http.MethodGet, target, nil)
		request.Header.Set("Authorization", "Bearer test-access-token")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want %d", target, response.Code, http.StatusBadRequest)
		}
	}
}

// eventFeedEmptyHandler builds the real mux over an empty in-memory event
// store, returning the store so a test can append while a long-poll holds.
func eventFeedEmptyHandler(t *testing.T) (http.Handler, *event.MemoryStore) {
	t.Helper()
	store := event.NewMemoryStore()
	runtime := DefaultRuntimeState(map[string]bool{})
	runtime.EventStore = store
	return NewWithRuntimeState(testStaticFiles(), testAuthService(), testVerifier{}, testOrganizationService{}, testTaskService{}, testSubmissionService{}, testLedgerService{}, testAgentService{}, testOrgCredentialService{}, testAssetService{}, runtime), store
}

func TestEventFeedLongPollReturnsEarlyWhenAnEventArrives(t *testing.T) {
	handler, store := eventFeedEmptyHandler(t)

	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = store.Append(context.Background(), testFeedEvent(t, event.KindTaskOpened), event.NewRecipients(stableTestUserID))
	}()

	started := time.Now()
	// httptest.ResponseRecorder implements http.Flusher, so the handler holds.
	body := fetchEventFeed(t, handler, "/api/events?wait=10")
	elapsed := time.Since(started)

	if len(body.Events) != 1 {
		t.Fatalf("long poll returned %d events, want 1: %+v", len(body.Events), body)
	}
	if elapsed >= 5*time.Second {
		t.Fatalf("long poll held for %v, want an early return after the event landed", elapsed)
	}
	if body.NextCursor == "" {
		t.Fatalf("long poll returned no resume cursor: %+v", body)
	}
}

func TestEventFeedLongPollReturnsEmptyWhenTheWaitElapses(t *testing.T) {
	handler, _ := eventFeedEmptyHandler(t)

	started := time.Now()
	body := fetchEventFeed(t, handler, "/api/events?wait=1")
	elapsed := time.Since(started)

	if len(body.Events) != 0 || body.NextCursor != "" {
		t.Fatalf("elapsed long poll = %+v, want an empty page", body)
	}
	if elapsed < time.Second {
		t.Fatalf("long poll returned after %v, want it to hold for the requested second", elapsed)
	}
}

func TestEventFeedLongPollDegradesWithoutFlusher(t *testing.T) {
	handler, _ := eventFeedEmptyHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/events?wait=10", nil)
	request.Header.Set("Authorization", "Bearer test-access-token")
	recorder := httptest.NewRecorder()
	started := time.Now()
	// The WASI guest bridge cannot hold requests; the transport probe must
	// turn the long poll into an immediate empty response.
	handler.ServeHTTP(nonFlushingRecorder{recorder: recorder}, request)
	elapsed := time.Since(started)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (%s)", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if elapsed >= 2*time.Second {
		t.Fatalf("guest-hosted long poll held for %v, want an immediate response", elapsed)
	}
	var body decodedEventListResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Events) != 0 || body.NextCursor != "" {
		t.Fatalf("degraded long poll = %+v, want an empty page", body)
	}
}

// ---- agent and org credential callers ----

// agentTokenRejectingVerifier refuses bearer tokens carrying the personal
// agent credential prefix, matching production where an agent secret is not
// a JWT, so requests fall through to the agent credential path.
type agentTokenRejectingVerifier struct{}

func (agentTokenRejectingVerifier) Verify(token auth.AccessToken) auth.SubjectVerifyResult {
	if agent.HasSecretPrefix(token.String()) {
		return auth.SubjectVerifyRejected{Reason: core.NewDomainError(core.ErrorCodeUnauthenticated, "not a user access token")}
	}
	return auth.SubjectVerified{Value: auth.UserSubject{ID: stableTestUserID}}
}

// stubAgentService verifies every agent secret as one fixed credential.
type stubAgentService struct {
	credential agent.Credential
}

func (service stubAgentService) Create(context.Context, core.UserID, agent.Label, agent.ScopeSet, *time.Time, *core.TaskID) agent.CreateResult {
	return agent.CreateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (service stubAgentService) Verify(context.Context, agent.SecretPlain) agent.VerifyResult {
	return agent.CredentialVerified{Subject: auth.UserSubject{ID: service.credential.UserID}, Credential: service.credential}
}

func (service stubAgentService) List(context.Context, core.UserID, core.Page) agent.ListResult {
	return agent.CredentialsListed{Values: []agent.Credential{}}
}

func (service stubAgentService) Revoke(context.Context, core.UserID, core.AgentCredentialID) agent.RevokeResult {
	return agent.RevokeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (service stubAgentService) ConfigureWorkPolicy(context.Context, core.UserID, core.AgentCredentialID, agent.WorkPolicy) agent.ConfigureWorkPolicyResult {
	return agent.ConfigureWorkPolicyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (service stubAgentService) WorkActivity(context.Context, core.UserID) agent.WorkActivityResult {
	return agent.WorkActivityListed{Values: []agent.CredentialWorkActivity{}}
}

// eventFeedAgentHandler builds the mux so a personal agent credential owned
// by stableTestUserID authenticates, with one event visible to that owner.
func eventFeedAgentHandler(t *testing.T, scopes []agent.Scope) (http.Handler, string) {
	t.Helper()
	store := event.NewMemoryStore()
	if _, matched := store.Append(context.Background(), testFeedEvent(t, event.KindTaskOpened), event.NewRecipients(stableTestUserID)).(event.AppendStoreAccepted); !matched {
		t.Fatalf("seed owner event rejected")
	}
	credential := agent.Credential{
		ID:     core.NewAgentCredentialID().(core.AgentCredentialIDCreated).Value,
		UserID: stableTestUserID,
		Label:  agent.NewLabel("Feed poller").(agent.LabelAccepted).Value,
		Scopes: agent.NewScopeSet(scopes),
		State:  agent.StateActive,
	}
	secret := agent.NewSecretPlain().(agent.SecretPlainAccepted).Value
	runtime := DefaultRuntimeState(map[string]bool{})
	runtime.EventStore = store
	handler := NewWithRuntimeState(testStaticFiles(), testAuthService(), agentTokenRejectingVerifier{}, testOrganizationService{}, testTaskService{}, testSubmissionService{}, testLedgerService{}, stubAgentService{credential: credential}, testOrgCredentialService{}, testAssetService{}, runtime)
	return handler, secret.String()
}

func TestEventFeedServesPersonalAgentCredentialWithNotificationsRead(t *testing.T) {
	handler, secret := eventFeedAgentHandler(t, []agent.Scope{agent.ScopeNotificationsRead})

	request := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	request.Header.Set("Authorization", "Bearer "+secret)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusOK, response.Body.String())
	}
	var body decodedEventListResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Events) != 1 || body.Events[0].Kind != "task_opened" {
		t.Fatalf("agent credential feed = %+v, want the owner's event", body)
	}
}

func TestEventFeedRejectsAgentCredentialMissingNotificationsRead(t *testing.T) {
	handler, secret := eventFeedAgentHandler(t, []agent.Scope{agent.ScopeTasksRead})

	request := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	request.Header.Set("Authorization", "Bearer "+secret)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusForbidden, response.Body.String())
	}
}

// eventFeedOrgHandler builds the mux so an org credential authenticates,
// with one event whose subject organization is that org and one for another
// organization.
func eventFeedOrgHandler(t *testing.T, scopes []agent.Scope) (http.Handler, string) {
	t.Helper()
	organizationID := core.NewOrganizationID().(core.OrganizationIDCreated).Value
	otherOrganizationID := core.NewOrganizationID().(core.OrganizationIDCreated).Value
	store := event.NewMemoryStore()
	for _, subjectOrganization := range []core.OrganizationID{organizationID, otherOrganizationID} {
		value := testFeedEvent(t, event.KindTaskOpened)
		value.Subject.Organization = event.OrganizationSubject{ID: subjectOrganization}
		if _, matched := store.Append(context.Background(), value, event.NewRecipients(stableTestUserID)).(event.AppendStoreAccepted); !matched {
			t.Fatalf("seed org event rejected")
		}
	}
	credential := orgcred.Credential{
		ID:             core.NewOrgCredentialID().(core.OrgCredentialIDCreated).Value,
		OrganizationID: organizationID,
		Label:          agent.NewLabel("Org feed poller").(agent.LabelAccepted).Value,
		Scopes:         agent.NewScopeSet(scopes),
		State:          agent.StateActive,
	}
	secret := orgcred.NewSecretPlain().(orgcred.SecretPlainAccepted).Value
	runtime := DefaultRuntimeState(map[string]bool{})
	runtime.EventStore = store
	handler := NewWithRuntimeState(testStaticFiles(), testAuthService(), orgTokenRejectingVerifier{}, testOrganizationService{}, testTaskService{}, testSubmissionService{}, testLedgerService{}, testAgentService{}, stubOrgCredentialService{credential: credential}, testAssetService{}, runtime)
	return handler, secret.String()
}

func TestEventFeedServesOrgCredentialItsOrganizationsEvents(t *testing.T) {
	handler, secret := eventFeedOrgHandler(t, []agent.Scope{agent.ScopeNotificationsRead})

	request := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	request.Header.Set("Authorization", "Bearer "+secret)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusOK, response.Body.String())
	}
	var body decodedEventListResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Events) != 1 {
		t.Fatalf("org credential feed = %+v, want exactly the credential's organization's event", body)
	}
}

func TestEventFeedRejectsOrgCredentialMissingNotificationsRead(t *testing.T) {
	handler, secret := eventFeedOrgHandler(t, []agent.Scope{agent.ScopeTasksRead})

	request := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	request.Header.Set("Authorization", "Bearer "+secret)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusForbidden, response.Body.String())
	}
}

func TestEventStreamStaysUserSessionOnly(t *testing.T) {
	handler, secret := eventFeedAgentHandler(t, []agent.Scope{agent.ScopeNotificationsRead})

	request := httptest.NewRequest(http.MethodGet, "/api/events/stream", nil)
	request.Header.Set("Authorization", "Bearer "+secret)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("stream status = %d, want %d (agents poll the feed instead)", response.Code, http.StatusUnauthorized)
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
