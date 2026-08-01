package notification

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
)

func TestNotifySkipsSelfNotification(t *testing.T) {
	user := newUserID(t)
	service := NewService(NewMemoryStore())

	result := service.Notify(context.Background(), user, user, KindSubmissionCreated, Subject{Kind: "submission", ID: "submission-1"}, EmptyMetadata(), NoSourceEvent{})
	if _, skipped := result.(NotificationSkipped); !skipped {
		t.Fatalf("expected self notification to be skipped, got %T", result)
	}

	listed := service.List(context.Background(), user, AnyState{}, core.DefaultPage()).(NotificationsListed)
	if len(listed.Values) != 0 {
		t.Fatalf("expected no self-notification rows, got %d", len(listed.Values))
	}
}

func TestNotifyListAndMarkRead(t *testing.T) {
	recipient := newUserID(t)
	actor := newUserID(t)
	service := NewService(NewMemoryStore())

	result := service.Notify(context.Background(), recipient, actor, KindSubmissionAccepted, Subject{Kind: "submission", ID: "submission-1"}, Metadata{JSON: `{"task_id":"task-1"}`}, NoSourceEvent{})
	created, matched := result.(NotificationCreated)
	if !matched {
		t.Fatalf("notify rejected: %T", result)
	}

	listed := service.List(context.Background(), recipient, AnyState{}, core.DefaultPage()).(NotificationsListed)
	if len(listed.Values) != 1 {
		t.Fatalf("expected one notification, got %d", len(listed.Values))
	}
	if listed.Values[0].State != StateUnread {
		t.Fatalf("expected unread state, got %s", listed.Values[0].State.String())
	}

	readResult := service.MarkRead(context.Background(), recipient, created.Value.ID)
	read, readMatched := readResult.(NotificationRead)
	if !readMatched {
		t.Fatalf("mark read rejected: %T", readResult)
	}
	if read.Value.State != StateRead {
		t.Fatalf("expected read state, got %s", read.Value.State.String())
	}
}

func newUserID(t *testing.T) core.UserID {
	t.Helper()
	created, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	return created.Value
}

func TestParseKindRoundTripsEveryKindAndRejectsUnknown(t *testing.T) {
	for _, kind := range AllKinds() {
		parsed, matched := ParseKind(kind.String()).(KindParsed)
		if !matched || parsed.Value != kind {
			t.Fatalf("ParseKind(%q) did not round trip", kind.String())
		}
	}
	rejected, matched := ParseKind("mystery_kind").(KindRejected)
	if !matched {
		t.Fatalf("unknown kind was parsed")
	}
	if rejected.Reason.Code() != core.ErrorCodeInvalidEnum {
		t.Fatalf("unknown kind rejection code = %s, want invalid_enum", rejected.Reason.Code())
	}
}

func TestParseStateRoundTripsAndRejectsUnknown(t *testing.T) {
	for _, state := range []State{StateUnread, StateRead} {
		parsed, matched := ParseState(state.String()).(StateParsed)
		if !matched || parsed.Value != state {
			t.Fatalf("ParseState(%q) did not round trip", state.String())
		}
	}
	rejected, matched := ParseState("mystery_state").(StateRejected)
	if !matched {
		t.Fatalf("unknown state was parsed")
	}
	if rejected.Reason.Code() != core.ErrorCodeInvalidEnum {
		t.Fatalf("unknown state rejection code = %s, want invalid_enum", rejected.Reason.Code())
	}
}

func TestListUnreadFilterAndCountUnread(t *testing.T) {
	recipient := newUserID(t)
	actor := newUserID(t)
	service := NewService(NewMemoryStore())

	first := service.Notify(context.Background(), recipient, actor, KindSubmissionCreated, Subject{Kind: "submission", ID: "submission-1"}, EmptyMetadata(), NoSourceEvent{})
	created, matched := first.(NotificationCreated)
	if !matched {
		t.Fatalf("first notify rejected: %T", first)
	}
	if _, matched := service.Notify(context.Background(), recipient, actor, KindTaskFunded, Subject{Kind: "task", ID: "task-1"}, EmptyMetadata(), NoSourceEvent{}).(NotificationCreated); !matched {
		t.Fatalf("second notify rejected")
	}
	if _, matched := service.MarkRead(context.Background(), recipient, created.Value.ID).(NotificationRead); !matched {
		t.Fatalf("mark read rejected")
	}

	unread := service.List(context.Background(), recipient, UnreadOnly{}, core.DefaultPage()).(NotificationsListed)
	if len(unread.Values) != 1 {
		t.Fatalf("unread listing returned %d rows, want 1", len(unread.Values))
	}
	if unread.Values[0].Kind != KindTaskFunded {
		t.Fatalf("unread listing returned kind %s", unread.Values[0].Kind.String())
	}

	all := service.List(context.Background(), recipient, AnyState{}, core.DefaultPage()).(NotificationsListed)
	if len(all.Values) != 2 {
		t.Fatalf("full listing returned %d rows, want 2", len(all.Values))
	}

	counted, matched := service.CountUnread(context.Background(), recipient).(UnreadCounted)
	if !matched || counted.Count != 1 {
		t.Fatalf("unread count = %+v, want 1", counted)
	}
	other, matched := service.CountUnread(context.Background(), actor).(UnreadCounted)
	if !matched || other.Count != 0 {
		t.Fatalf("other user's unread count = %+v, want 0", other)
	}
}
