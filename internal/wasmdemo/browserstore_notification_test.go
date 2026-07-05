package wasmdemo

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/notification"
)

func testUserID(t *testing.T, label string) core.UserID {
	t.Helper()
	result := core.NewUserID()
	created, matched := result.(core.UserIDCreated)
	if !matched {
		t.Fatalf("new user id for %q failed", label)
	}
	return created.Value
}

func testPage(t *testing.T, limit int, offset int) core.Page {
	t.Helper()
	result := core.NewPage(limit, offset)
	accepted, matched := result.(core.PageAccepted)
	if !matched {
		t.Fatalf("new page (%d, %d) failed", limit, offset)
	}
	return accepted.Value
}

func TestNotificationBrowserStoreCreateAndList(t *testing.T) {
	store := NewNotificationBrowserStore(newTestBrowserStorage())
	service := notification.NewService(store)
	ctx := context.Background()
	recipient := testUserID(t, "user-recipient")
	actor := testUserID(t, "user-actor")

	for index := 0; index < 3; index++ {
		result := service.Notify(ctx, recipient, actor, notification.KindSubmissionCreated, notification.Subject{Kind: "task", ID: "task-1"}, notification.EmptyMetadata())
		if _, matched := result.(notification.NotificationCreated); !matched {
			t.Fatalf("notify %d: want NotificationCreated, got %#v", index, result)
		}
	}

	listResult := service.List(ctx, recipient, testPage(t, 10, 0))
	listed, matched := listResult.(notification.NotificationsListed)
	if !matched {
		t.Fatalf("list: want NotificationsListed, got %#v", listResult)
	}
	if len(listed.Values) != 3 {
		t.Fatalf("list count = %d, want 3", len(listed.Values))
	}
	for _, value := range listed.Values {
		if value.State != notification.StateUnread {
			t.Fatalf("new notification state = %q, want unread", value.State.String())
		}
	}
}

func TestNotificationBrowserStoreListOrdersNewestFirst(t *testing.T) {
	store := NewNotificationBrowserStore(newTestBrowserStorage())
	service := notification.NewService(store)
	ctx := context.Background()
	recipient := testUserID(t, "user-recipient")
	actor := testUserID(t, "user-actor")

	var createdIDs []string
	for index := 0; index < 3; index++ {
		result := service.Notify(ctx, recipient, actor, notification.KindSubmissionCreated, notification.Subject{Kind: "task", ID: "task-1"}, notification.EmptyMetadata())
		created, matched := result.(notification.NotificationCreated)
		if !matched {
			t.Fatalf("notify %d failed: %#v", index, result)
		}
		createdIDs = append(createdIDs, created.Value.ID.String())
	}

	listResult := service.List(ctx, recipient, testPage(t, 10, 0))
	listed := listResult.(notification.NotificationsListed)
	if len(listed.Values) != 3 {
		t.Fatalf("list count = %d, want 3", len(listed.Values))
	}
	// Newest first: the last one created should be first in the list.
	if listed.Values[0].ID.String() != createdIDs[2] {
		t.Fatalf("first listed id = %q, want the most recently created %q", listed.Values[0].ID.String(), createdIDs[2])
	}
	if listed.Values[2].ID.String() != createdIDs[0] {
		t.Fatalf("last listed id = %q, want the first created %q", listed.Values[2].ID.String(), createdIDs[0])
	}
}

func TestNotificationBrowserStoreListPaginates(t *testing.T) {
	store := NewNotificationBrowserStore(newTestBrowserStorage())
	service := notification.NewService(store)
	ctx := context.Background()
	recipient := testUserID(t, "user-recipient")
	actor := testUserID(t, "user-actor")

	for index := 0; index < 5; index++ {
		service.Notify(ctx, recipient, actor, notification.KindSubmissionCreated, notification.Subject{Kind: "task", ID: "task-1"}, notification.EmptyMetadata())
	}

	firstPage := service.List(ctx, recipient, testPage(t, 2, 0)).(notification.NotificationsListed)
	if len(firstPage.Values) != 2 {
		t.Fatalf("first page count = %d, want 2", len(firstPage.Values))
	}
	secondPage := service.List(ctx, recipient, testPage(t, 2, 2)).(notification.NotificationsListed)
	if len(secondPage.Values) != 2 {
		t.Fatalf("second page count = %d, want 2", len(secondPage.Values))
	}
	thirdPage := service.List(ctx, recipient, testPage(t, 2, 4)).(notification.NotificationsListed)
	if len(thirdPage.Values) != 1 {
		t.Fatalf("third page count = %d, want 1", len(thirdPage.Values))
	}
	if firstPage.Values[0].ID == secondPage.Values[0].ID {
		t.Fatalf("first and second page overlap on id %v", firstPage.Values[0].ID)
	}
}

func TestNotificationBrowserStoreListIsScopedToRecipient(t *testing.T) {
	store := NewNotificationBrowserStore(newTestBrowserStorage())
	service := notification.NewService(store)
	ctx := context.Background()
	recipientA := testUserID(t, "user-a")
	recipientB := testUserID(t, "user-b")
	actor := testUserID(t, "user-actor")

	service.Notify(ctx, recipientA, actor, notification.KindSubmissionCreated, notification.Subject{Kind: "task", ID: "task-1"}, notification.EmptyMetadata())
	service.Notify(ctx, recipientB, actor, notification.KindSubmissionCreated, notification.Subject{Kind: "task", ID: "task-2"}, notification.EmptyMetadata())

	listA := service.List(ctx, recipientA, testPage(t, 10, 0)).(notification.NotificationsListed)
	if len(listA.Values) != 1 {
		t.Fatalf("recipient A list count = %d, want 1", len(listA.Values))
	}
	if listA.Values[0].RecipientID != recipientA {
		t.Fatalf("recipient A got a notification addressed to %v", listA.Values[0].RecipientID)
	}
}

func TestNotificationBrowserStoreMarkRead(t *testing.T) {
	store := NewNotificationBrowserStore(newTestBrowserStorage())
	service := notification.NewService(store)
	ctx := context.Background()
	recipient := testUserID(t, "user-recipient")
	actor := testUserID(t, "user-actor")

	created := service.Notify(ctx, recipient, actor, notification.KindSubmissionCreated, notification.Subject{Kind: "task", ID: "task-1"}, notification.EmptyMetadata()).(notification.NotificationCreated)

	markResult := service.MarkRead(ctx, recipient, created.Value.ID)
	read, matched := markResult.(notification.NotificationRead)
	if !matched {
		t.Fatalf("mark read: want NotificationRead, got %#v", markResult)
	}
	if read.Value.State != notification.StateRead {
		t.Fatalf("marked notification state = %q, want read", read.Value.State.String())
	}

	listResult := service.List(ctx, recipient, testPage(t, 10, 0)).(notification.NotificationsListed)
	if listResult.Values[0].State != notification.StateRead {
		t.Fatalf("listed notification state after mark-read = %q, want read", listResult.Values[0].State.String())
	}
}

func TestNotificationBrowserStoreMarkReadRejectsWrongRecipient(t *testing.T) {
	store := NewNotificationBrowserStore(newTestBrowserStorage())
	service := notification.NewService(store)
	ctx := context.Background()
	recipient := testUserID(t, "user-recipient")
	other := testUserID(t, "user-other")
	actor := testUserID(t, "user-actor")

	created := service.Notify(ctx, recipient, actor, notification.KindSubmissionCreated, notification.Subject{Kind: "task", ID: "task-1"}, notification.EmptyMetadata()).(notification.NotificationCreated)

	markResult := service.MarkRead(ctx, other, created.Value.ID)
	if _, matched := markResult.(notification.MarkReadRejected); !matched {
		t.Fatalf("mark read by wrong recipient: want MarkReadRejected, got %#v", markResult)
	}
}

func TestNotificationBrowserStoreMarkReadRejectsUnknownID(t *testing.T) {
	store := NewNotificationBrowserStore(newTestBrowserStorage())
	service := notification.NewService(store)
	ctx := context.Background()
	recipient := testUserID(t, "user-recipient")

	unknownIDResult := core.NewNotificationID()
	unknownID := unknownIDResult.(core.NotificationIDCreated).Value

	markResult := service.MarkRead(ctx, recipient, unknownID)
	if _, matched := markResult.(notification.MarkReadRejected); !matched {
		t.Fatalf("mark read unknown id: want MarkReadRejected, got %#v", markResult)
	}
}
