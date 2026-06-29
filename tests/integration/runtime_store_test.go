//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/notification"
)

func TestNotificationStorePersistsInboxLifecycle(t *testing.T) {
	pool := newPool(t)
	recipient := createUser(t, pool, "notification-recipient")
	actor := createUser(t, pool, "notification-actor")
	submissionID := newSubmissionID(t)
	taskID := newTaskID(t)

	service := notification.NewService(db.NewNotificationStore(pool))
	result := service.Notify(context.Background(), recipient, actor, notification.KindSubmissionCreated, notification.Subject{Kind: "submission", ID: submissionID.String()}, notification.Metadata{JSON: `{"task_id":"` + taskID.String() + `"}`})
	created, matched := result.(notification.NotificationCreated)
	if !matched {
		t.Fatalf("notify rejected: %T", result)
	}

	listResult := service.List(context.Background(), recipient, core.DefaultPage())
	listed, listedMatched := listResult.(notification.NotificationsListed)
	if !listedMatched {
		t.Fatalf("list notifications rejected: %T", listResult)
	}
	if len(listed.Values) != 1 {
		t.Fatalf("expected one notification, got %d", len(listed.Values))
	}
	if listed.Values[0].State != notification.StateUnread {
		t.Fatalf("expected unread state, got %s", listed.Values[0].State.String())
	}

	readResult := service.MarkRead(context.Background(), recipient, created.Value.ID)
	read, readMatched := readResult.(notification.NotificationRead)
	if !readMatched {
		t.Fatalf("mark read rejected: %T", readResult)
	}
	if read.Value.State != notification.StateRead {
		t.Fatalf("expected read state, got %s", read.Value.State.String())
	}

	selfResult := service.Notify(context.Background(), recipient, recipient, notification.KindSubmissionAccepted, notification.Subject{Kind: "submission", ID: submissionID.String()}, notification.EmptyMetadata())
	if _, skipped := selfResult.(notification.NotificationSkipped); !skipped {
		t.Fatalf("self notification should be skipped, got %T", selfResult)
	}
	secondList := service.List(context.Background(), recipient, core.DefaultPage()).(notification.NotificationsListed)
	if len(secondList.Values) != 1 {
		t.Fatalf("self notification created a row")
	}
}

func TestAuditStoreListsPersistedEvents(t *testing.T) {
	pool := newPool(t)
	actor := createUser(t, pool, "audit-actor")
	service := audit.NewService(db.NewAuditStore(pool))

	recordResult := service.Record(context.Background(), actor, audit.ActionSubmissionAccepted, audit.Subject{Kind: "submission", ID: newSubmissionID(t).String()}, audit.Metadata{JSON: "{}"})
	if _, recorded := recordResult.(audit.EventRecorded); !recorded {
		t.Fatalf("record audit rejected: %T", recordResult)
	}

	listResult := service.List(context.Background(), audit.NoListFilters(), core.DefaultPage())
	listed, listedMatched := listResult.(audit.EventsListed)
	if !listedMatched {
		t.Fatalf("list audit rejected: %T", listResult)
	}
	if len(listed.Values) == 0 {
		t.Fatalf("expected at least one audit event")
	}
	if listed.Values[0].ActorUserID != actor {
		t.Fatalf("expected latest audit actor %s, got %s", actor.String(), listed.Values[0].ActorUserID.String())
	}

	filteredResult := service.List(context.Background(), audit.ListFilters{Action: audit.ActionEquals{Value: audit.ActionSubmissionAccepted}, SubjectKind: audit.SubjectKindEquals{Value: "submission"}, SubjectID: audit.AnySubjectID{}}, core.DefaultPage())
	filtered, filteredMatched := filteredResult.(audit.EventsListed)
	if !filteredMatched {
		t.Fatalf("filtered list audit rejected: %T", filteredResult)
	}
	if len(filtered.Values) == 0 {
		t.Fatalf("expected filtered audit event")
	}
}

func TestPrivacyStoreResolvesExportAndSensitiveRedaction(t *testing.T) {
	pool := newPool(t)
	requester := createUser(t, pool, "privacy-requester")
	taskID := insertTask(t, pool, requester, "open", 1)
	submissionID := insertSubmission(t, pool, taskID, requester)
	if _, err := pool.Exec(context.Background(), `
		insert into submission_sensitive_fields (submission_id, field_index, path, category, retention, redaction)
		values ($1, 0, 'email', 'pii', 'delete_on_request', 'replace')
	`, submissionID.String()); err != nil {
		t.Fatalf("insert sensitive field: %v", err)
	}

	store := db.NewPrivacyStore(pool)
	exportResult := store.Create(context.Background(), requester, "data_export")
	exportCreated, exportMatched := exportResult.(httpserver.PrivacyRequestSaved)
	if !exportMatched {
		t.Fatalf("create export rejected: %T", exportResult)
	}
	resolvedExportResult := store.Resolve(context.Background(), exportCreated.Value.ID, "export generated")
	resolvedExport, resolvedExportMatched := resolvedExportResult.(httpserver.PrivacyRequestSaved)
	if !resolvedExportMatched {
		if rejected, rejectedMatched := resolvedExportResult.(httpserver.PrivacyRequestMutationRejected); rejectedMatched {
			t.Fatalf("resolve export rejected: %s", rejected.Reason.Description())
		}
		t.Fatalf("resolve export rejected: %T", resolvedExportResult)
	}
	var exportBody struct {
		UserID      string `json:"user_id"`
		Submissions []struct {
			ID string `json:"id"`
		} `json:"submissions"`
		SensitiveFields []struct {
			State string `json:"state"`
		} `json:"sensitive_fields"`
	}
	if err := json.Unmarshal([]byte(resolvedExport.Value.ExportJSON), &exportBody); err != nil {
		t.Fatalf("decode export json: %v", err)
	}
	if exportBody.UserID != requester.String() {
		t.Fatalf("export user_id = %q, want %s", exportBody.UserID, requester.String())
	}
	if len(exportBody.Submissions) != 1 {
		t.Fatalf("export submissions = %d, want 1", len(exportBody.Submissions))
	}
	if len(exportBody.SensitiveFields) != 1 || exportBody.SensitiveFields[0].State != "active" {
		t.Fatalf("export sensitive fields = %#v, want one active field", exportBody.SensitiveFields)
	}

	deletionResult := store.Create(context.Background(), requester, "sensitive_field_deletion")
	deletionCreated, deletionMatched := deletionResult.(httpserver.PrivacyRequestSaved)
	if !deletionMatched {
		t.Fatalf("create deletion rejected: %T", deletionResult)
	}
	resolvedDeletionResult := store.Resolve(context.Background(), deletionCreated.Value.ID, "fields redacted")
	resolvedDeletion, resolvedDeletionMatched := resolvedDeletionResult.(httpserver.PrivacyRequestSaved)
	if !resolvedDeletionMatched {
		if rejected, rejectedMatched := resolvedDeletionResult.(httpserver.PrivacyRequestMutationRejected); rejectedMatched {
			t.Fatalf("resolve deletion rejected: %s", rejected.Reason.Description())
		}
		t.Fatalf("resolve deletion rejected: %T", resolvedDeletionResult)
	}
	if resolvedDeletion.Value.RedactedFieldCount != 1 {
		t.Fatalf("redacted count = %d, want 1", resolvedDeletion.Value.RedactedFieldCount)
	}
	var fieldState string
	var redactedAt time.Time
	if err := pool.QueryRow(context.Background(), `
		select state, redacted_at
		from submission_sensitive_fields
		where submission_id = $1 and path = 'email'
	`, submissionID.String()).Scan(&fieldState, &redactedAt); err != nil {
		t.Fatalf("read sensitive field: %v", err)
	}
	if fieldState != "redacted" {
		t.Fatalf("field state = %q, want redacted", fieldState)
	}
	if redactedAt.IsZero() {
		t.Fatalf("redacted_at is zero")
	}
	var eventCount int
	if err := pool.QueryRow(context.Background(), `
		select count(*)
		from submission_sensitive_field_events
		where submission_id = $1 and action = 'sensitive_field_redacted'
	`, submissionID.String()).Scan(&eventCount); err != nil {
		t.Fatalf("count sensitive field events: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("redaction event count = %d, want 1", eventCount)
	}
}

func TestMCPSessionStoreCountsActiveSessions(t *testing.T) {
	pool := newPool(t)
	store := db.NewMCPSessionStore(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	subject := "integration-" + newUserID(t).String()

	if err := store.CreateMCPSession(ctx, subject+"-a", subject, now); err != nil {
		t.Fatalf("create session a: %v", err)
	}
	if err := store.CreateMCPSession(ctx, subject+"-b", subject, now.Add(time.Second)); err != nil {
		t.Fatalf("create session b: %v", err)
	}
	count, err := store.ActiveMCPSessionCountForSubject(ctx, subject, now.Add(-time.Minute))
	if err != nil {
		t.Fatalf("count subject sessions: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected two active sessions, got %d", count)
	}

	closed, err := store.CloseMCPSession(ctx, subject+"-a", now.Add(2*time.Second))
	if err != nil {
		t.Fatalf("close session: %v", err)
	}
	if !closed {
		t.Fatalf("expected session close to affect a row")
	}
	count, err = store.ActiveMCPSessionCountForSubject(ctx, subject, now.Add(-time.Minute))
	if err != nil {
		t.Fatalf("count subject sessions after close: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one active session after close, got %d", count)
	}

	eventID, payload, err := store.AppendMCPEvent(ctx, subject+"-b", []byte(`{"jsonrpc":"2.0"}`), now.Add(3*time.Second))
	if err != nil {
		t.Fatalf("append session event: %v", err)
	}
	if eventID == "" || string(payload) != `{"jsonrpc":"2.0"}` {
		t.Fatalf("unexpected persisted event: %q %s", eventID, string(payload))
	}
	eventIDs, payloads, err := store.ListMCPEvents(ctx, subject+"-b", "", 100)
	if err != nil {
		t.Fatalf("list session events: %v", err)
	}
	if len(eventIDs) != 1 || eventIDs[0] != eventID || string(payloads[0]) != `{"jsonrpc":"2.0"}` {
		t.Fatalf("persisted MCP events did not round trip")
	}
	afterIDs, _, err := store.ListMCPEvents(ctx, subject+"-b", eventID, 100)
	if err != nil {
		t.Fatalf("list session events after event id: %v", err)
	}
	if len(afterIDs) != 0 {
		t.Fatalf("expected no replay events after the last event id")
	}
}

func TestPostgresRateLimiterPersistsBuckets(t *testing.T) {
	pool := newPool(t)
	prefix := "integration-" + newUserID(t).String()
	limiter := db.NewRateLimiter(pool, prefix, 1, 1)

	if !limiter.Allow("client") {
		t.Fatalf("first request should be allowed")
	}
	if limiter.Allow("client") {
		t.Fatalf("second request should be rate limited")
	}
	if limiter.ActiveBuckets() != 1 {
		t.Fatalf("expected one persisted rate-limit bucket")
	}
}

func newSubmissionID(t *testing.T) core.SubmissionID {
	t.Helper()
	created, matched := core.NewSubmissionID().(core.SubmissionIDCreated)
	if !matched {
		t.Fatalf("submission id rejected")
	}
	return created.Value
}
