package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/sqlitex"
)

// openSQLiteWithSchema opens an ncruces SQLite database and applies the real
// Postgres migrations translated to the SQLite dialect. This exercises the DDL
// translation against every migration file.
func openSQLiteWithSchema(t *testing.T) *sql.DB {
	t.Helper()
	handle, err := sqlitex.Open("file:" + t.TempDir() + "/demo.db?_pragma=foreign_keys(off)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = handle.Close() })
	if err := MigrateUpSQLite(context.Background(), handle, os.DirFS("../../migrations")); err != nil {
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

// seedUserForTest inserts a minimal users row (the enriched read models join
// users for display names) and returns its id.
func seedUserForTest(t *testing.T, handle *sql.DB, name string) core.UserID {
	t.Helper()
	id := newUserIDForTest(t)
	if _, err := handle.ExecContext(context.Background(),
		"insert into users (id, email, display_name) values (?, ?, ?)",
		id.String(), name+"-"+id.String()+"@example.test", name); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}

// TestNewSchemaDDLOnSQLite proves the newer migrations translate to SQLite:
// the bigserial event cursor auto-assigns monotonically (rowid alias), ledger
// idempotency keys are unique per account rather than globally, the outbox
// dispatch-state column defaults to recorded (000046) while the old webhook
// pump cursor table is gone, the system actor row is seeded, and the task
// expires_at column exists.
func TestNewSchemaDDLOnSQLite(t *testing.T) {
	ctx := context.Background()
	handle := openSQLiteWithSchema(t)

	var firstSeq, secondSeq int64
	if err := handle.QueryRowContext(ctx,
		"insert into domain_events (id, kind, actor_kind) values ('event-1', 'task_opened', 'system') returning seq",
	).Scan(&firstSeq); err != nil {
		t.Fatalf("insert first event: %v", err)
	}
	if err := handle.QueryRowContext(ctx,
		"insert into domain_events (id, kind, actor_kind) values ('event-2', 'task_funded', 'system') returning seq",
	).Scan(&secondSeq); err != nil {
		t.Fatalf("insert second event: %v", err)
	}
	if secondSeq <= firstSeq {
		t.Fatalf("event seq not monotonic: first %d, second %d", firstSeq, secondSeq)
	}

	insertLedgerRow := func(rowID, accountID, key string) error {
		_, err := handle.ExecContext(ctx,
			"insert into ledger_entries (id, account_id, kind, amount, idempotency_key) values ($1, $2, 'task_escrow', 5, $3)",
			rowID, accountID, key,
		)
		return err
	}
	if err := insertLedgerRow("ledger-1", "account-1", "retry-1"); err != nil {
		t.Fatalf("first ledger row: %v", err)
	}
	if err := insertLedgerRow("ledger-2", "account-2", "retry-1"); err != nil {
		t.Fatalf("same key on another account must be allowed: %v", err)
	}
	if err := insertLedgerRow("ledger-3", "account-1", "retry-1"); err == nil {
		t.Fatalf("same key on the same account must be rejected")
	}

	// The outbox migration (000046) adds the dispatch-state enum with a
	// 'recorded' default and drops the old webhook pump cursor table.
	var dispatchState string
	if err := handle.QueryRowContext(ctx,
		"select dispatch_state from domain_events where id = 'event-1'",
	).Scan(&dispatchState); err != nil {
		t.Fatalf("read dispatch state: %v", err)
	}
	if dispatchState != "recorded" {
		t.Fatalf("dispatch_state = %q, want recorded", dispatchState)
	}
	var pumpCursorExists int
	if err := handle.QueryRowContext(ctx,
		"select count(*) from sqlite_master where type = 'table' and name = 'webhook_pump_cursor'",
	).Scan(&pumpCursorExists); err != nil {
		t.Fatalf("check pump cursor table: %v", err)
	}
	if pumpCursorExists != 0 {
		t.Fatalf("webhook_pump_cursor must be dropped by the outbox migration")
	}

	var systemEmail string
	if err := handle.QueryRowContext(ctx,
		"select email from users where id = '00000000-0000-7000-8000-000000000001'",
	).Scan(&systemEmail); err != nil {
		t.Fatalf("system actor row not seeded: %v", err)
	}
	if systemEmail != "system@sharecrop.invalid" {
		t.Fatalf("system actor email = %q", systemEmail)
	}

	if _, err := handle.ExecContext(ctx,
		"update tasks set expires_at = $1 where 1 = 0", time.Now().UTC(),
	); err != nil {
		t.Fatalf("tasks.expires_at missing: %v", err)
	}
}

// TestEventStoreOnSQLite proves the event store's transactional append,
// recipient fan-out, and cursor-filtered listing through the SQLite dialect.
func TestEventStoreOnSQLite(t *testing.T) {
	ctx := context.Background()
	handle := openSQLiteWithSchema(t)
	store := EventStore{db: NewSQLite(handle)}

	actor := seedUserForTest(t, handle, "Feed Actor")
	owner := newUserIDForTest(t)
	stranger := newUserIDForTest(t)
	taskID, _ := core.NewTaskID().(core.TaskIDCreated)

	emit := func(kind event.Kind) event.StoredEvent {
		t.Helper()
		idResult, _ := core.NewDomainEventID().(core.DomainEventIDCreated)
		subject := event.NoSubjectRefs()
		subject.Task = event.TaskSubject{ID: taskID.Value}
		appended, matched := store.Append(ctx, event.Event{
			ID:         idResult.Value,
			Kind:       kind,
			Actor:      event.ActorUser{ID: actor},
			Subject:    subject,
			Metadata:   event.EmptyMetadata(),
			OccurredAt: time.Now().UTC().Round(time.Microsecond),
		}, event.NewRecipients(actor, owner)).(event.AppendStoreAccepted)
		if !matched {
			t.Fatalf("append rejected")
		}
		return appended.Value
	}

	first := emit(event.KindTaskFunded)
	second := emit(event.KindSubmissionCreated)
	if second.Cursor.Sequence() <= first.Cursor.Sequence() {
		t.Fatalf("cursors not monotonic: %d then %d", first.Cursor.Sequence(), second.Cursor.Sequence())
	}

	page, _ := core.NewPage(50, 0).(core.PageAccepted)
	listed, matched := store.ListForRecipient(ctx, owner, event.FromStart{}, page.Value).(event.ListStoreAccepted)
	if !matched {
		t.Fatalf("list rejected")
	}
	if len(listed.Values) != 2 {
		t.Fatalf("owner feed has %d events, want 2", len(listed.Values))
	}
	got := listed.Values[0]
	if got.Event.Kind != event.KindTaskFunded {
		t.Fatalf("first event kind = %q", got.Event.Kind.String())
	}
	if ref, ok := got.Event.Subject.Task.(event.TaskSubject); !ok || ref.ID != taskID.Value {
		t.Fatalf("task ref did not round-trip")
	}
	if _, ok := got.Event.Subject.Submission.(event.NoSubmission); !ok {
		t.Fatalf("absent submission ref did not round-trip as absent")
	}
	if event.ActorUserID(got.Event.Actor) != actor {
		t.Fatalf("actor did not round-trip")
	}

	after, matched := store.ListForRecipient(ctx, owner, event.After{Cursor: first.Cursor}, page.Value).(event.ListStoreAccepted)
	if !matched {
		t.Fatalf("after-cursor list rejected")
	}
	if len(after.Values) != 1 || after.Values[0].Event.Kind != event.KindSubmissionCreated {
		t.Fatalf("after-cursor feed wrong: %d values", len(after.Values))
	}

	strangerFeed, matched := store.ListForRecipient(ctx, stranger, event.FromStart{}, page.Value).(event.ListStoreAccepted)
	if !matched {
		t.Fatalf("stranger list rejected")
	}
	if len(strangerFeed.Values) != 0 {
		t.Fatalf("stranger must not see the events, got %d", len(strangerFeed.Values))
	}
}

// TestNotificationStoreOnSQLite runs the real notification store against SQLite
// through the handle abstraction, proving the statement dialect (now(),
// ::casts, returning, $N placeholders) and the timestamptz time round-trip.
func TestNotificationStoreOnSQLite(t *testing.T) {
	ctx := context.Background()
	handle := openSQLiteWithSchema(t)
	store := NotificationStore{db: NewSQLite(handle)}

	recipient := newUserIDForTest(t)
	actor := seedUserForTest(t, handle, "Inbox Actor")
	notificationID, matched := core.NewNotificationID().(core.NotificationIDCreated)
	if !matched {
		t.Fatalf("notification id rejected")
	}
	createdAt := time.Now().UTC().Round(time.Microsecond)
	value := notification.Notification{
		ID:           notificationID.Value,
		RecipientID:  recipient,
		ActorID:      actor,
		Kind:         notification.KindSubmissionAccepted,
		Subject:      notification.Subject{Kind: "submission", ID: "sub-1"},
		SubjectTitle: notification.NoSubjectTitle{},
		State:        notification.StateUnread,
		Metadata:     notification.EmptyMetadata(),
		CreatedAt:    createdAt,
	}

	if _, ok := store.Create(ctx, value).(notification.CreateStoreAccepted); !ok {
		t.Fatalf("create rejected")
	}

	page, ok := core.NewPage(50, 0).(core.PageAccepted)
	if !ok {
		t.Fatalf("page rejected")
	}
	listed, ok := store.List(ctx, recipient, notification.AnyState{}, page.Value).(notification.ListStoreAccepted)
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

// TestSplitPartTranslation proves the Postgres email-local-part expression the
// display-name reads use becomes a valid SQLite substr/instr expression.
func TestSplitPartTranslation(t *testing.T) {
	got := translateSQLiteStatement("select split_part(users.email, '@', 1) from users")
	want := "select substr(users.email, 1, instr(users.email, '@') - 1) from users"
	if got != want {
		t.Fatalf("translated = %q, want %q", got, want)
	}
}
