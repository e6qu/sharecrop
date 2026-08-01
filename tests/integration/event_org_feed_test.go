//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
)

// appendOrganizationEvent appends one event whose subject organization is
// the given org, acted by the given user, and returns the stored row.
func appendOrganizationEvent(t *testing.T, store db.EventStore, actor core.UserID, organizationID core.OrganizationID) event.StoredEvent {
	t.Helper()
	idCreated, matched := core.NewDomainEventID().(core.DomainEventIDCreated)
	if !matched {
		t.Fatalf("domain event id rejected")
	}
	subject := event.NoSubjectRefs()
	subject.Organization = event.OrganizationSubject{ID: organizationID}
	appended, appendMatched := store.Append(context.Background(), event.Event{
		ID:       idCreated.Value,
		Kind:     event.KindCreditGranted,
		Actor:    event.ActorUser{ID: actor},
		Subject:  subject,
		Metadata: event.EmptyMetadata(),
	}, event.NewRecipients(actor)).(event.AppendStoreAccepted)
	if !appendMatched {
		t.Fatalf("append organization event rejected")
	}
	return appended.Value
}

// TestEventStoreListsForOrganization pins the org-scoped cursor feed read:
// only events whose subject organization matches are listed, the cursor
// filter excludes already-seen rows, and paging applies.
func TestEventStoreListsForOrganization(t *testing.T) {
	pool := newPool(t)
	store := db.NewEventStore(pool)
	actor := createUser(t, pool, "org-feed-actor")

	organizationID := core.NewOrganizationID().(core.OrganizationIDCreated).Value
	otherOrganizationID := core.NewOrganizationID().(core.OrganizationIDCreated).Value

	first := appendOrganizationEvent(t, store, actor, organizationID)
	appendOrganizationEvent(t, store, actor, otherOrganizationID)
	second := appendOrganizationEvent(t, store, actor, organizationID)

	listed, matched := store.ListForOrganization(context.Background(), organizationID, event.FromStart{}, core.DefaultPage()).(event.ListStoreAccepted)
	if !matched {
		t.Fatalf("list for organization rejected")
	}
	if len(listed.Values) != 2 {
		t.Fatalf("organization feed rows = %d, want 2", len(listed.Values))
	}
	if listed.Values[0].Cursor != first.Cursor || listed.Values[1].Cursor != second.Cursor {
		t.Fatalf("organization feed cursors = %v/%v, want %v/%v",
			listed.Values[0].Cursor, listed.Values[1].Cursor, first.Cursor, second.Cursor)
	}

	afterFirst, afterMatched := store.ListForOrganization(context.Background(), organizationID, event.After{Cursor: first.Cursor}, core.DefaultPage()).(event.ListStoreAccepted)
	if !afterMatched {
		t.Fatalf("list after cursor rejected")
	}
	if len(afterFirst.Values) != 1 || afterFirst.Values[0].Cursor != second.Cursor {
		t.Fatalf("after-cursor feed = %+v, want only the second event", afterFirst.Values)
	}

	pageResult, pageMatched := core.NewPage(1, 0).(core.PageAccepted)
	if !pageMatched {
		t.Fatalf("page rejected")
	}
	limited, limitedMatched := store.ListForOrganization(context.Background(), organizationID, event.FromStart{}, pageResult.Value).(event.ListStoreAccepted)
	if !limitedMatched {
		t.Fatalf("limited list rejected")
	}
	if len(limited.Values) != 1 || limited.Values[0].Cursor != first.Cursor {
		t.Fatalf("limited feed = %+v, want only the first event", limited.Values)
	}
}
