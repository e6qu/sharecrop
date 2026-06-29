package notification

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
)

func TestNotifySkipsSelfNotification(t *testing.T) {
	user := newUserID(t)
	service := NewService(NewMemoryStore())

	result := service.Notify(context.Background(), user, user, KindSubmissionCreated, Subject{Kind: "submission", ID: "submission-1"}, EmptyMetadata())
	if _, skipped := result.(NotificationSkipped); !skipped {
		t.Fatalf("expected self notification to be skipped, got %T", result)
	}

	listed := service.List(context.Background(), user, core.DefaultPage()).(NotificationsListed)
	if len(listed.Values) != 0 {
		t.Fatalf("expected no self-notification rows, got %d", len(listed.Values))
	}
}

func TestNotifyListAndMarkRead(t *testing.T) {
	recipient := newUserID(t)
	actor := newUserID(t)
	service := NewService(NewMemoryStore())

	result := service.Notify(context.Background(), recipient, actor, KindSubmissionAccepted, Subject{Kind: "submission", ID: "submission-1"}, Metadata{JSON: `{"task_id":"task-1"}`})
	created, matched := result.(NotificationCreated)
	if !matched {
		t.Fatalf("notify rejected: %T", result)
	}

	listed := service.List(context.Background(), recipient, core.DefaultPage()).(NotificationsListed)
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
