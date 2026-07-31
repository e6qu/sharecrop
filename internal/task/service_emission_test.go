package task

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/event/eventtest"
)

// emissionHarness wires a task service over the capturing event store so each
// test can assert exactly which events a mutation emitted.
type emissionHarness struct {
	store    *taskMemoryStore
	events   *eventtest.CapturingStore
	service  Service
	owner    Task
	ownerRaw core.UserID
}

func newEmissionHarness(t *testing.T) emissionHarness {
	t.Helper()
	store := newTaskMemoryStore()
	events := eventtest.NewCapturingStore()
	service := NewService(store, newTaskPermissionStore(), nil, eventtest.RecorderOver(events))
	actor := testUserSubject(t)
	created, matched := service.Create(context.Background(), testCreateCommand(t, actor, UserOwner{UserID: actor.ID}, PublicVisibility{})).(TaskCreated)
	if !matched {
		t.Fatalf("create rejected")
	}
	return emissionHarness{store: store, events: events, service: service, owner: created.Value, ownerRaw: actor.ID}
}

func testUserSubjectFor(userID core.UserID) auth.UserSubject {
	return auth.UserSubject{ID: userID}
}

func lastEvent(t *testing.T, events *eventtest.CapturingStore) event.Event {
	t.Helper()
	appended := events.Appended()
	if len(appended) == 0 {
		t.Fatalf("no events were emitted")
	}
	return appended[len(appended)-1]
}

func recipientsContain(recipients event.Recipients, user core.UserID) bool {
	for _, value := range recipients.Users {
		if value == user {
			return true
		}
	}
	return false
}

func TestOpenEmitsTaskOpenedToOwner(t *testing.T) {
	harness := newEmissionHarness(t)
	if _, matched := harness.service.Open(context.Background(), testUserSubjectFor(harness.ownerRaw), harness.owner.ID).(TaskStateChanged); !matched {
		t.Fatalf("open rejected")
	}
	emitted := lastEvent(t, harness.events)
	if emitted.Kind != event.KindTaskOpened {
		t.Fatalf("emitted kind = %s, want task_opened", emitted.Kind.String())
	}
	if _, matched := emitted.Subject.Task.(event.TaskSubject); !matched {
		t.Fatalf("task_opened event has no task subject")
	}
	if !recipientsContain(harness.events.RecipientsAt(len(harness.events.Appended())-1), harness.ownerRaw) {
		t.Fatalf("task_opened recipients missed the owner")
	}
}

func TestReserveEmitsReservationRequestedToOwnerAndWorker(t *testing.T) {
	harness := newEmissionHarness(t)
	if _, matched := harness.service.Open(context.Background(), testUserSubjectFor(harness.ownerRaw), harness.owner.ID).(TaskStateChanged); !matched {
		t.Fatalf("open rejected")
	}
	worker := testUserSubject(t)
	reserved, matched := harness.service.Reserve(context.Background(), worker, harness.owner.ID).(ReservationCreated)
	if !matched {
		t.Fatalf("reserve rejected")
	}

	emitted := lastEvent(t, harness.events)
	if emitted.Kind != event.KindReservationRequested {
		t.Fatalf("emitted kind = %s, want reservation_requested", emitted.Kind.String())
	}
	subjectReservation, matched := emitted.Subject.Reservation.(event.ReservationSubject)
	if !matched || subjectReservation.ID != reserved.Value.ID {
		t.Fatalf("reservation subject did not carry the created reservation")
	}
	recipients := harness.events.RecipientsAt(len(harness.events.Appended()) - 1)
	if !recipientsContain(recipients, harness.ownerRaw) || !recipientsContain(recipients, worker.ID) {
		t.Fatalf("reservation_requested recipients = %v, want owner and worker", recipients.Users)
	}
}

func TestCancelEmitsTaskCancelledToOwnerAndActiveHolder(t *testing.T) {
	harness := newEmissionHarness(t)
	ownerSubject := testUserSubjectFor(harness.ownerRaw)
	if _, matched := harness.service.Open(context.Background(), ownerSubject, harness.owner.ID).(TaskStateChanged); !matched {
		t.Fatalf("open rejected")
	}
	worker := testUserSubject(t)
	if _, matched := harness.service.Reserve(context.Background(), worker, harness.owner.ID).(ReservationCreated); !matched {
		t.Fatalf("reserve rejected")
	}

	if _, matched := harness.service.Cancel(context.Background(), ownerSubject, harness.owner.ID).(TaskStateChanged); !matched {
		t.Fatalf("cancel rejected")
	}
	emitted := lastEvent(t, harness.events)
	if emitted.Kind != event.KindTaskCancelled {
		t.Fatalf("emitted kind = %s, want task_cancelled", emitted.Kind.String())
	}
	recipients := harness.events.RecipientsAt(len(harness.events.Appended()) - 1)
	if !recipientsContain(recipients, harness.ownerRaw) || !recipientsContain(recipients, worker.ID) {
		t.Fatalf("task_cancelled recipients = %v, want owner and active holder", recipients.Users)
	}
}

func TestAddTaskCommentEmitsTaskCommentedToBothParties(t *testing.T) {
	harness := newEmissionHarness(t)
	ownerSubject := testUserSubjectFor(harness.ownerRaw)
	if _, matched := harness.service.Open(context.Background(), ownerSubject, harness.owner.ID).(TaskStateChanged); !matched {
		t.Fatalf("open rejected")
	}
	worker := testUserSubject(t)
	if _, matched := harness.service.Reserve(context.Background(), worker, harness.owner.ID).(ReservationCreated); !matched {
		t.Fatalf("reserve rejected")
	}

	body, matched := NewCommentBody("Does the schema allow markdown?").(CommentBodyAccepted)
	if !matched {
		t.Fatalf("comment body rejected")
	}
	if _, matched := harness.service.AddTaskComment(context.Background(), ownerSubject, harness.owner.ID, body.Value).(TaskCommentAdded); !matched {
		t.Fatalf("comment rejected")
	}
	emitted := lastEvent(t, harness.events)
	if emitted.Kind != event.KindTaskCommented {
		t.Fatalf("emitted kind = %s, want task_commented", emitted.Kind.String())
	}
	recipients := harness.events.RecipientsAt(len(harness.events.Appended()) - 1)
	if !recipientsContain(recipients, harness.ownerRaw) || !recipientsContain(recipients, worker.ID) {
		t.Fatalf("task_commented recipients = %v, want owner and active holder", recipients.Users)
	}
}
