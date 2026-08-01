package event

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/notification"
)

func newUserID(t *testing.T) core.UserID {
	t.Helper()
	created, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	return created.Value
}

func TestParseKindRoundTripsEveryKind(t *testing.T) {
	kinds := AllKinds()
	if len(kinds) != 23 {
		t.Fatalf("AllKinds() has %d kinds, want 23", len(kinds))
	}
	for _, kind := range kinds {
		parsed, matched := ParseKind(kind.String()).(KindParsed)
		if !matched {
			t.Fatalf("ParseKind(%q) rejected", kind.String())
		}
		if parsed.Value != kind {
			t.Fatalf("ParseKind(%q) = %q", kind.String(), parsed.Value.String())
		}
	}
	if _, matched := ParseKind("task_exploded").(KindRejected); !matched {
		t.Fatalf("unknown kind must be rejected")
	}
}

func TestNotificationRuleIsTotalAndFeedOnlyKindsAreExplicit(t *testing.T) {
	feedOnly := map[string]bool{KindTaskOpened.String(): true}
	for _, kind := range AllKinds() {
		rule := NotificationRuleFor(kind)
		switch typed := rule.(type) {
		case NoNotification:
			if !feedOnly[kind.String()] {
				t.Fatalf("kind %q unexpectedly produces no notification", kind.String())
			}
		case NotifyAs:
			if typed.Kind.String() == "" {
				t.Fatalf("kind %q maps to an empty notification kind", kind.String())
			}
		default:
			t.Fatalf("kind %q has no notification rule", kind.String())
		}
	}
}

func TestParseCursor(t *testing.T) {
	parsed, matched := ParseCursor("42").(CursorParsed)
	if !matched {
		t.Fatalf("ParseCursor(42) rejected")
	}
	if parsed.Value.Sequence() != 42 || parsed.Value.String() != "42" {
		t.Fatalf("cursor round-trip failed: %d %q", parsed.Value.Sequence(), parsed.Value.String())
	}
	for _, raw := range []string{"", "abc", "-1", "1.5"} {
		if _, rejected := ParseCursor(raw).(CursorRejected); !rejected {
			t.Fatalf("ParseCursor(%q) must be rejected", raw)
		}
	}
}

func TestNewRecipientsDeduplicates(t *testing.T) {
	first := newUserID(t)
	second := newUserID(t)
	recipients := NewRecipients(first, second, first)
	if len(recipients.Users) != 2 {
		t.Fatalf("recipients = %d, want 2", len(recipients.Users))
	}
}

func TestActorUserIDResolvesSystem(t *testing.T) {
	user := newUserID(t)
	if ActorUserID(ActorUser{ID: user}) != user {
		t.Fatalf("ActorUser must resolve to its own id")
	}
	if ActorUserID(ActorSystem{}) != core.SystemUserID() {
		t.Fatalf("ActorSystem must resolve to the system actor")
	}
}

type fakeEventStore struct {
	appended   []Event
	recipients []Recipients
	dispatched []core.DomainEventID
}

func (store *fakeEventStore) Append(_ context.Context, value Event, recipients Recipients) AppendStoreResult {
	store.appended = append(store.appended, value)
	store.recipients = append(store.recipients, recipients)
	return AppendStoreAccepted{Value: WithoutEnrichment(StoredEvent{Event: value, Cursor: CursorFromSequence(int64(len(store.appended)))})}
}

func (store *fakeEventStore) Dispatch(_ context.Context, id core.DomainEventID) DispatchStoreResult {
	store.dispatched = append(store.dispatched, id)
	return DispatchStoreCompleted{}
}

func (store *fakeEventStore) ListForRecipient(context.Context, core.UserID, CursorFilter, core.Page) ListStoreResult {
	return ListStoreAccepted{Values: nil}
}

func (store *fakeEventStore) ListForOrganization(context.Context, core.OrganizationID, CursorFilter, core.Page) ListStoreResult {
	return ListStoreAccepted{Values: nil}
}

type fakeNotificationStore struct {
	created []notification.Notification
	failing bool
}

func (store *fakeNotificationStore) Create(_ context.Context, value notification.Notification) notification.CreateStoreResult {
	if store.failing {
		return notification.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "injected notification store failure")}
	}
	store.created = append(store.created, value)
	return notification.CreateStoreAccepted{}
}

func (store *fakeNotificationStore) List(context.Context, core.UserID, notification.StateFilter, core.Page) notification.ListStoreResult {
	return notification.ListStoreAccepted{}
}

func (store *fakeNotificationStore) CountUnread(context.Context, core.UserID) notification.CountStoreResult {
	return notification.CountUnreadCounted{Count: int64(len(store.created))}
}

func (store *fakeNotificationStore) MarkRead(context.Context, core.UserID, core.NotificationID) notification.MarkReadStoreResult {
	return notification.MarkReadStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "not stored")}
}

func mustDraft(t *testing.T, kind Kind, actor Actor, subject Subject, recipients Recipients) Draft {
	t.Helper()
	created, matched := NewDraft(kind, actor, subject, EmptyMetadata(), recipients).(DraftCreated)
	if !matched {
		t.Fatalf("draft rejected")
	}
	return created.Value
}

func TestDispatchFansOutToRecipientsExceptActor(t *testing.T) {
	eventStore := &fakeEventStore{}
	notificationStore := &fakeNotificationStore{}
	recorder := NewRecorder(eventStore, notification.NewService(notificationStore))

	actor := newUserID(t)
	owner := newUserID(t)
	taskID, _ := core.NewTaskID().(core.TaskIDCreated)
	subject := NoSubjectRefs()
	subject.Task = TaskSubject{ID: taskID.Value}
	draft := mustDraft(t, KindReservationRequested, ActorUser{ID: actor}, subject, NewRecipients(actor, owner))

	recorder.Dispatch(context.Background(), draft)

	// notification.Service skips the actor, so only the owner gets an inbox row.
	if len(notificationStore.created) != 1 {
		t.Fatalf("notifications created = %d, want 1", len(notificationStore.created))
	}
	created := notificationStore.created[0]
	if created.RecipientID != owner || created.ActorID != actor {
		t.Fatalf("notification recipient/actor wrong")
	}
	if created.Kind != notification.KindReservationRequested {
		t.Fatalf("notification kind = %q", created.Kind.String())
	}
	if created.Subject.Kind != "task" || created.Subject.ID != taskID.Value.String() {
		t.Fatalf("notification subject = %+v", created.Subject)
	}
	if _, sourced := created.Source.(notification.FromEvent); !sourced {
		t.Fatalf("notification must be keyed by its source event")
	}
	if len(eventStore.dispatched) != 1 || eventStore.dispatched[0] != draft.ID {
		t.Fatalf("dispatch must complete the store-side step for the draft")
	}
}

func TestDispatchFeedOnlyKindCreatesNoNotification(t *testing.T) {
	eventStore := &fakeEventStore{}
	notificationStore := &fakeNotificationStore{}
	recorder := NewRecorder(eventStore, notification.NewService(notificationStore))

	actor := newUserID(t)
	draft := mustDraft(t, KindTaskOpened, ActorUser{ID: actor}, NoSubjectRefs(), NewRecipients(actor))
	recorder.Dispatch(context.Background(), draft)
	if len(notificationStore.created) != 0 {
		t.Fatalf("feed-only kind must not notify")
	}
	if len(eventStore.dispatched) != 1 {
		t.Fatalf("feed-only kind must still be marked dispatched")
	}
}

// TestDispatchLeavesRowRecordedWhenFanOutFails is the recorder half of the
// unrecoverable-loss fix: a failed inbox fan-out must not flip the event to
// dispatched, so the recovery sweep retries the whole idempotent step later.
func TestDispatchLeavesRowRecordedWhenFanOutFails(t *testing.T) {
	eventStore := &fakeEventStore{}
	notificationStore := &fakeNotificationStore{failing: true}
	recorder := NewRecorder(eventStore, notification.NewService(notificationStore))

	actor := newUserID(t)
	owner := newUserID(t)
	taskID, _ := core.NewTaskID().(core.TaskIDCreated)
	subject := NoSubjectRefs()
	subject.Task = TaskSubject{ID: taskID.Value}
	draft := mustDraft(t, KindReservationExpired, ActorUser{ID: actor}, subject, NewRecipients(actor, owner))

	recorder.Dispatch(context.Background(), draft)
	if len(eventStore.dispatched) != 0 {
		t.Fatalf("failed fan-out must leave the event recorded, got %d store dispatches", len(eventStore.dispatched))
	}

	// The failure clears; the retried dispatch completes both legs.
	notificationStore.failing = false
	recorder.Dispatch(context.Background(), draft)
	if len(notificationStore.created) != 1 {
		t.Fatalf("retried fan-out created %d notifications, want 1", len(notificationStore.created))
	}
	if len(eventStore.dispatched) != 1 {
		t.Fatalf("retried dispatch must complete the store-side step")
	}
}

func TestNotificationSubjectPrefersMostSpecificRef(t *testing.T) {
	submissionID, _ := core.NewSubmissionID().(core.SubmissionIDCreated)
	taskID, _ := core.NewTaskID().(core.TaskIDCreated)
	subject := NoSubjectRefs()
	subject.Task = TaskSubject{ID: taskID.Value}
	subject.Submission = SubmissionSubject{ID: submissionID.Value}
	mapped := NotificationSubjectFor(subject)
	if mapped.Kind != "submission" || mapped.ID != submissionID.Value.String() {
		t.Fatalf("subject = %+v, want submission ref", mapped)
	}
}
