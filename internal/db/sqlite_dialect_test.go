package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/sqlitex"
)

// openSQLiteWithSchema opens an ncruces SQLite database and applies the real
// Postgres migrations translated to the SQLite dialect. This exercises the DDL
// translation against all 34 migration files.
func openSQLiteWithSchema(t *testing.T) *sql.DB {
	t.Helper()
	handle, err := sqlitex.Open("file:" + t.TempDir() + "/demo.db?_pragma=foreign_keys(off)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = handle.Close() })
	if err := MigrateUpSQLite(context.Background(), handle, "../../migrations"); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	return handle
}

func newUserIDForTest(t *testing.T) core.UserID {
	t.Helper()
	created, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	return created.Value
}

// TestNotificationStoreOnSQLite runs the real notification store against SQLite
// through the handle abstraction, proving the statement dialect (now(),
// ::casts, returning, $N placeholders) and the timestamptz time round-trip.
func TestNotificationStoreOnSQLite(t *testing.T) {
	ctx := context.Background()
	store := NotificationStore{db: NewSQLite(openSQLiteWithSchema(t))}

	recipient := newUserIDForTest(t)
	actor := newUserIDForTest(t)
	notificationID, matched := core.NewNotificationID().(core.NotificationIDCreated)
	if !matched {
		t.Fatalf("notification id rejected")
	}
	createdAt := time.Now().UTC().Round(time.Microsecond)
	value := notification.Notification{
		ID:          notificationID.Value,
		RecipientID: recipient,
		ActorID:     actor,
		Kind:        notification.KindSubmissionAccepted,
		Subject:     notification.Subject{Kind: "submission", ID: "sub-1"},
		State:       notification.StateUnread,
		Metadata:    notification.EmptyMetadata(),
		CreatedAt:   createdAt,
	}

	if _, ok := store.Create(ctx, value).(notification.CreateStoreAccepted); !ok {
		t.Fatalf("create rejected")
	}

	page, ok := core.NewPage(50, 0).(core.PageAccepted)
	if !ok {
		t.Fatalf("page rejected")
	}
	listed, ok := store.List(ctx, recipient, page.Value).(notification.ListStoreAccepted)
	if !ok {
		t.Fatalf("list rejected")
	}
	if len(listed.Values) != 1 {
		t.Fatalf("listed %d notifications, want 1", len(listed.Values))
	}
	got := listed.Values[0]
	if got.ID != value.ID {
		t.Fatalf("id = %s, want %s", got.ID, value.ID)
	}
	if !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("created_at = %s, want %s (timestamp round-trip failed)", got.CreatedAt, createdAt)
	}
	if got.State != notification.StateUnread {
		t.Fatalf("state = %v, want unread", got.State)
	}

	marked, ok := store.MarkRead(ctx, recipient, value.ID).(notification.MarkReadStoreAccepted)
	if !ok {
		t.Fatalf("mark read rejected")
	}
	if marked.Value.State != notification.StateRead {
		t.Fatalf("state after mark read = %v, want read", marked.Value.State)
	}
}
