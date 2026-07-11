package notificationbridge

import (
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/notification/notificationtest"
)

func sampleNotification(t *testing.T) notification.Notification {
	t.Helper()
	id, matched := core.NewNotificationID().(core.NotificationIDCreated)
	if !matched {
		t.Fatalf("notification id rejected")
	}
	recipient, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("recipient id rejected")
	}
	actor, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("actor id rejected")
	}
	return notification.Notification{
		ID:          id.Value,
		RecipientID: recipient.Value,
		ActorID:     actor.Value,
		Kind:        notification.KindSubmissionAccepted,
		Subject:     notification.Subject{Kind: "submission", ID: "sub-1"},
		State:       notification.StateUnread,
		Metadata:    notification.EmptyMetadata(),
		CreatedAt:   time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC),
	}
}

func assertNotificationEqual(t *testing.T, got, want notification.Notification) {
	t.Helper()
	if diff := notificationtest.NotificationDiff(got, want); diff != "" {
		t.Errorf("notification mismatch: %s", diff)
	}
}

func TestNotificationRoundTrip(t *testing.T) {
	original := sampleNotification(t)
	restored, err := decodeNotification(encodeNotification(original))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertNotificationEqual(t, restored, original)
}

func TestCreateResultRoundTrip(t *testing.T) {
	accepted, err := decodeCreateResult(encodeCreateResult(notification.CreateStoreAccepted{}))
	if err != nil {
		t.Fatalf("decode accepted: %v", err)
	}
	if _, matched := accepted.(notification.CreateStoreAccepted); !matched {
		t.Fatalf("accepted create result = %T", accepted)
	}

	rejected, err := decodeCreateResult(encodeCreateResult(notification.CreateStoreRejected{
		Reason: core.NewDomainError(core.ErrorCodeConflict, "duplicate"),
	}))
	if err != nil {
		t.Fatalf("decode rejected: %v", err)
	}
	typed, matched := rejected.(notification.CreateStoreRejected)
	if !matched || typed.Reason.Code() != core.ErrorCodeConflict {
		t.Errorf("create rejection not preserved: %T", rejected)
	}
}

func TestListResultRoundTrip(t *testing.T) {
	first := sampleNotification(t)
	second := sampleNotification(t)
	listed, err := decodeListResult(encodeListResult(notification.ListStoreAccepted{Values: []notification.Notification{first, second}}))
	if err != nil {
		t.Fatalf("decode listed: %v", err)
	}
	typed, matched := listed.(notification.ListStoreAccepted)
	if !matched || len(typed.Values) != 2 {
		t.Fatalf("listed result = %T", listed)
	}
	assertNotificationEqual(t, typed.Values[0], first)
	assertNotificationEqual(t, typed.Values[1], second)
}

func TestMarkReadResultRoundTrip(t *testing.T) {
	value := sampleNotification(t)
	accepted, err := decodeMarkReadResult(encodeMarkReadResult(notification.MarkReadStoreAccepted{Value: value}))
	if err != nil {
		t.Fatalf("decode accepted: %v", err)
	}
	typed, matched := accepted.(notification.MarkReadStoreAccepted)
	if !matched {
		t.Fatalf("accepted mark-read result = %T", accepted)
	}
	assertNotificationEqual(t, typed.Value, value)

	rejected, err := decodeMarkReadResult(encodeMarkReadResult(notification.MarkReadStoreRejected{
		Reason: core.NewDomainError(core.ErrorCodeNotFound, "missing"),
	}))
	if err != nil {
		t.Fatalf("decode rejected: %v", err)
	}
	if typed, matched := rejected.(notification.MarkReadStoreRejected); !matched || typed.Reason.Code() != core.ErrorCodeNotFound {
		t.Errorf("mark-read rejection not preserved: %T", rejected)
	}
}
