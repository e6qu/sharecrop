package db

import (
	"context"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	httpserver "github.com/e6qu/sharecrop/internal/http"
)

func requirePageForTest(t *testing.T, limit, offset int) core.Page {
	t.Helper()
	page, ok := core.NewPage(limit, offset).(core.PageAccepted)
	if !ok {
		t.Fatalf("page rejected")
	}
	return page.Value
}

func newAuditEventIDForTest(t *testing.T) core.AuditEventID {
	t.Helper()
	created, ok := core.NewAuditEventID().(core.AuditEventIDCreated)
	if !ok {
		t.Fatalf("audit event id rejected")
	}
	return created.Value
}

// TestAuditStoreOnSQLite exercises the NamedArgs path (@limit/@offset/@action)
// and strftime-ordered listing against SQLite.
func TestAuditStoreOnSQLite(t *testing.T) {
	ctx := context.Background()
	store := AuditStore{db: NewSQLite(openSQLiteWithSchema(t))}
	actor := newUserIDForTest(t)

	funded := audit.Event{
		ID:          newAuditEventIDForTest(t),
		ActorUserID: actor,
		Action:      audit.ActionTaskFunded,
		Subject:     audit.Subject{Kind: "task", ID: "t1"},
		Metadata:    audit.EmptyMetadata(),
		CreatedAt:   time.Now().UTC().Add(-time.Minute).Round(time.Microsecond),
	}
	accepted := audit.Event{
		ID:          newAuditEventIDForTest(t),
		ActorUserID: actor,
		Action:      audit.ActionSubmissionAccepted,
		Subject:     audit.Subject{Kind: "submission", ID: "s1"},
		Metadata:    audit.EmptyMetadata(),
		CreatedAt:   time.Now().UTC().Round(time.Microsecond),
	}
	for _, event := range []audit.Event{funded, accepted} {
		if _, ok := store.Record(ctx, event).(audit.EventRecorded); !ok {
			t.Fatalf("record rejected")
		}
	}

	page := requirePageForTest(t, 50, 0)
	all, ok := store.List(ctx, audit.NoListFilters(), page).(audit.EventsListed)
	if !ok {
		t.Fatalf("list rejected")
	}
	if len(all.Values) != 2 {
		t.Fatalf("listed %d events, want 2", len(all.Values))
	}
	if all.Values[0].ID != accepted.ID {
		t.Fatalf("newest-first ordering wrong: got %s, want %s", all.Values[0].ID, accepted.ID)
	}

	filtered, ok := store.List(ctx, audit.ListFilters{
		Action:      audit.ActionEquals{Value: audit.ActionTaskFunded},
		SubjectKind: audit.AnySubjectKind{},
		SubjectID:   audit.AnySubjectID{},
	}, page).(audit.EventsListed)
	if !ok {
		t.Fatalf("filtered list rejected")
	}
	if len(filtered.Values) != 1 || filtered.Values[0].ID != funded.ID {
		t.Fatalf("action filter wrong: %d events", len(filtered.Values))
	}

	got, ok := store.Get(ctx, funded.ID).(audit.EventFound)
	if !ok {
		t.Fatalf("get rejected")
	}
	if !got.Value.CreatedAt.Equal(funded.CreatedAt) {
		t.Fatalf("get timestamp = %s, want %s", got.Value.CreatedAt, funded.CreatedAt)
	}
}

// TestSavedQueueViewUpsertOnSQLite exercises ON CONFLICT DO UPDATE + excluded on
// SQLite: a second upsert for the same (user, scope, name) updates in place.
func TestSavedQueueViewUpsertOnSQLite(t *testing.T) {
	ctx := context.Background()
	store := SavedQueueViewStore{db: NewSQLite(openSQLiteWithSchema(t))}
	user := newUserIDForTest(t)

	view := httpserver.SavedQueueView{
		UserID:      user,
		Scope:       "team_work",
		Name:        "mine",
		Query:       "state:open",
		StateFilter: "open",
		TypeFilter:  "",
		Sort:        "newest",
	}
	if _, ok := store.Upsert(ctx, view).(httpserver.SavedQueueViewSaved); !ok {
		t.Fatalf("first upsert rejected")
	}
	view.Query = "state:closed"
	if _, ok := store.Upsert(ctx, view).(httpserver.SavedQueueViewSaved); !ok {
		t.Fatalf("second upsert (on conflict) rejected")
	}

	listed, ok := store.List(ctx, user, "team_work").(httpserver.SavedQueueViewsListed)
	if !ok {
		t.Fatalf("list rejected")
	}
	if len(listed.Values) != 1 {
		t.Fatalf("listed %d views, want 1 (upsert must update, not insert)", len(listed.Values))
	}
	if listed.Values[0].Query != "state:closed" {
		t.Fatalf("query = %q, want updated to state:closed", listed.Values[0].Query)
	}
}
