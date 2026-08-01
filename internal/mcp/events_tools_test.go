package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
)

// ListEvents serves the fake feed, honoring the cursor filter and the page
// limit like the real store, so cursor threading is observable in tests.
func (services fakeServices) ListEvents(_ context.Context, _ auth.Subject, filter event.CursorFilter, page core.Page) event.ListStoreResult {
	values := make([]event.StoredEvent, 0, len(services.eventFeed))
	for _, stored := range services.eventFeed {
		if after, matched := filter.(event.After); matched && stored.Cursor.Sequence() <= after.Cursor.Sequence() {
			continue
		}
		if len(values) == page.Limit() {
			break
		}
		values = append(values, stored)
	}
	return event.ListStoreAccepted{Values: values}
}

func testStoredEvent(t *testing.T, sequence int64, kind event.Kind, actorID core.UserID, taskID core.TaskID, title string) event.StoredEvent {
	t.Helper()
	eventID, matched := core.NewDomainEventID().(core.DomainEventIDCreated)
	if !matched {
		t.Fatalf("domain event id rejected")
	}
	subject := event.NoSubjectRefs()
	subject.Task = event.TaskSubject{ID: taskID}
	return event.StoredEvent{
		Event: event.Event{
			ID:         eventID.Value,
			Kind:       kind,
			Actor:      event.ActorUser{ID: actorID},
			Subject:    subject,
			Metadata:   event.EmptyMetadata(),
			OccurredAt: time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC),
		},
		Cursor:    event.CursorFromSequence(sequence),
		ActorName: event.ActorNamed{DisplayName: auth.NewDisplayName("mara").(auth.DisplayNameAccepted).Value},
		TaskTitle: event.TaskTitled{Title: title},
	}
}

func notificationsReadScopes() agent.ScopeSet {
	return agent.NewScopeSet([]agent.Scope{agent.ScopeNotificationsRead})
}

func TestToolsCallListEventsReturnsRowsAndNextCursor(t *testing.T) {
	actorID := core.NewUserID().(core.UserIDCreated).Value
	taskID := core.NewTaskID().(core.TaskIDCreated).Value
	feed := []event.StoredEvent{
		testStoredEvent(t, 7, event.KindTaskFunded, actorID, taskID, "Review PR 7"),
		testStoredEvent(t, 9, event.KindSubmissionAccepted, actorID, taskID, "Review PR 7"),
	}
	server := NewServer(fakeServices{eventFeed: feed})
	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: notificationsReadScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.list_events","arguments":{}}`)))

	var payload struct {
		Events []struct {
			ID               string `json:"id"`
			Kind             string `json:"kind"`
			ActorID          string `json:"actor_id"`
			ActorDisplayName string `json:"actor_display_name"`
			SubjectKind      string `json:"subject_kind"`
			SubjectID        string `json:"subject_id"`
			TaskTitle        string `json:"task_title"`
			OccurredAt       string `json:"occurred_at"`
		} `json:"events"`
		NextCursor string `json:"next_cursor"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode events payload: %v (%s)", err, content)
	}
	if len(payload.Events) != 2 {
		t.Fatalf("events = %d, want 2 (%s)", len(payload.Events), content)
	}
	first := payload.Events[0]
	if first.Kind != "task_funded" || first.ActorID != actorID.String() || first.ActorDisplayName != "mara" {
		t.Fatalf("first event actor fields wrong: %+v", first)
	}
	if first.SubjectKind != "task" || first.SubjectID != taskID.String() || first.TaskTitle != "Review PR 7" {
		t.Fatalf("first event subject fields wrong: %+v", first)
	}
	if first.ID == "" || first.OccurredAt != "2026-07-30T12:00:00Z" {
		t.Fatalf("first event id/occurred_at wrong: %+v", first)
	}
	if payload.NextCursor != "9" {
		t.Fatalf("next_cursor = %q, want 9", payload.NextCursor)
	}
}

func TestToolsCallListEventsEmptyPageHasEmptyNextCursor(t *testing.T) {
	server := NewServer(fakeServices{})
	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: notificationsReadScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.list_events","arguments":{}}`)))
	if !strings.Contains(content, `"events":[]`) || !strings.Contains(content, `"next_cursor":""`) {
		t.Fatalf("empty feed payload wrong: %s", content)
	}
}

func TestToolsCallListEventsResumesAfterCursorAndHonorsLimit(t *testing.T) {
	actorID := core.NewUserID().(core.UserIDCreated).Value
	taskID := core.NewTaskID().(core.TaskIDCreated).Value
	feed := []event.StoredEvent{
		testStoredEvent(t, 3, event.KindTaskFunded, actorID, taskID, "A"),
		testStoredEvent(t, 5, event.KindTaskOpened, actorID, taskID, "A"),
		testStoredEvent(t, 8, event.KindSubmissionCreated, actorID, taskID, "A"),
	}
	server := NewServer(fakeServices{eventFeed: feed})
	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: notificationsReadScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.list_events","arguments":{"after":"3","limit":1}}`)))
	var payload struct {
		Events     []json.RawMessage `json:"events"`
		NextCursor string            `json:"next_cursor"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode events payload: %v", err)
	}
	if len(payload.Events) != 1 || payload.NextCursor != "5" {
		t.Fatalf("after+limit page = %d rows, next_cursor %q; want 1 row and cursor 5 (%s)", len(payload.Events), payload.NextCursor, content)
	}
}

func TestToolsCallListEventsRejectsMalformedCursor(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: notificationsReadScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.list_events","arguments":{"after":"not-a-cursor"}}`))
	if response.Error == nil || response.Error.Code != codeInvalidParams {
		t.Fatalf("expected invalid-params for a malformed cursor, got %+v", response.Error)
	}
}

func TestToolsCallListEventsRequiresNotificationsReadScope(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.list_events","arguments":{}}`))
	if response.Error == nil || response.Error.Code != codeScopeDenied {
		t.Fatalf("expected scope-denied without notifications_read, got %+v", response.Error)
	}
}

// TestToolsCallListEventsAcceptsOrganizationCredential pins that list_events
// dispatches on auth.Subject rather than requiring a personal credential: an
// organization subject reaches the service instead of failing the
// user-subject gate.
func TestToolsCallListEventsAcceptsOrganizationCredential(t *testing.T) {
	organizationID := core.NewOrganizationID().(core.OrganizationIDCreated).Value
	server := NewServer(fakeServices{})
	content := decodeToolText(t, server.Handle(context.Background(), auth.OrgSubject{ID: organizationID}, CallerCredential{Scopes: notificationsReadScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.list_events","arguments":{}}`)))
	if !strings.Contains(content, `"events"`) {
		t.Fatalf("org credential list_events content missing events key: %s", content)
	}
}
